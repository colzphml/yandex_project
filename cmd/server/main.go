package main

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/colzphml/yandex_project/internal/handlers"

	//"middleware" используется в 2 пакетах, потому для собственного алиас
	mdw "github.com/colzphml/yandex_project/internal/middleware"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
	log                 = zerolog.New(serverutils.LogConfig()).With().Timestamp().Str("component", "server").Logger()
)

func HTTPServer(ctx context.Context, cfg *serverutils.ServerConfig, repo storage.Repositorier) *http.Server {
	r := chi.NewRouter()
	r.Use(mdw.GzipHandle)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/update/{metric_type}/{metric_name}/{metric_value}", handlers.SaveHandler(ctx, repo, cfg))
	r.Get("/value/{metric_type}/{metric_name}", handlers.GetValueHandler(ctx, repo))
	r.Post("/update/", handlers.SaveJSONHandler(ctx, repo, cfg))
	r.Post("/updates/", handlers.SaveJSONArrayHandler(ctx, repo, cfg))
	r.Post("/value/", handlers.GetJSONValueHandler(ctx, repo, cfg))
	r.Get("/ping", handlers.PingHandler(ctx, repo, cfg))
	r.Get("/", handlers.ListMetricsHandler(ctx, repo, cfg))
	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("failed initialize server")
		}
	}()
	return srv
}

func main() {
	/*
		go func() {
			fmt.Println(http.ListenAndServe("localhost:6061", nil))
		}()
	*/
	log.Info().Msg("server started")
	log.Info().Msg("Build version: " + buildVersion)
	log.Info().Msg("Build date: " + buildDate)
	log.Info().Msg("Build commit: " + buildCommit)
	cfg := serverutils.LoadServerConfig()
	log.Info().Dict("cfg", zerolog.Dict().
		Str("ServerAddress", cfg.ServerAddress).
		Dur("StoreInterval", cfg.StoreInterval).
		Str("StoreFile", cfg.StoreFile).
		Bool("Restore", cfg.Restore).
		Str("Key", cfg.Key).
		Str("DSN", cfg.DBDSN),
	).Msg("Server config")
	ctx := context.Background()
	repo, tickerSave, err := storage.CreateRepo(ctx, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("create repo failed")
	}
	//для "штатного" завершения сервера
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	srv := HTTPServer(ctx, cfg, repo)
Loop:
	for {
		select {
		case <-tickerSave.C:
			err := repo.DumpMetrics(ctx, cfg)
			if err != nil {
				log.Error().Err(err).Msg("failed dump metrics")
			}
			log.Info().Msg("metrics stored by interval")
		case <-sigChan:
			ctxcancel, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer func() {
				repo.DumpMetrics(ctx, cfg)
				log.Info().Msg("metrics stored")
				repo.Close()
				tickerSave.Stop()
				cancel()
			}()
			if err := srv.Shutdown(ctxcancel); err != nil {
				log.Error().Err(err).Msg("failed shutdown server")
			}
			log.Info().Msg("server stopped")
			break Loop
		}
	}
}
