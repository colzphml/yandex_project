package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/colzphml/yandex_project/internal/handlers"
	//"middleware" используется в 2 пакетах, потому для собственного алиас
	mdw "github.com/colzphml/yandex_project/internal/middleware"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

//вынес в отдельную функцию создание сервера
func HTTPServer(cfg *serverutils.ServerConfig, repo storage.Repositorier) *http.Server {
	r := chi.NewRouter()
	r.Use(mdw.GzipHandle)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/update/{metric_type}/{metric_name}/{metric_value}", handlers.SaveHandler(repo, cfg))
	r.Get("/value/{metric_type}/{metric_name}", handlers.GetValueHandler(repo))
	r.Post("/update/", handlers.SaveJSONHandler(repo, cfg))
	r.Post("/updates/", handlers.SaveJSONArrayHandler(repo, cfg))
	r.Post("/value/", handlers.GetJSONValueHandler(repo, cfg))
	r.Get("/ping", handlers.PingHandler(repo, cfg))
	r.Get("/", handlers.ListMetricsHandler(repo, cfg))
	srv := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	return srv
}

func main() {
	cfg := serverutils.LoadServerConfig()
	log.Println(cfg)
	repo, tickerSave, err := storage.CreateRepo(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	//для "штатного" завершения сервера
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	srv := HTTPServer(cfg, repo)
Loop:
	for {
		select {
		case <-tickerSave.C:
			err := repo.DumpMetrics(cfg)
			if err != nil {
				log.Println(err.Error())
			}
		case <-sigChan:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer func() {
				repo.DumpMetrics(cfg)
				log.Println("metrics stored")
				repo.Close()
				tickerSave.Stop()
				cancel()
			}()
			if err := srv.Shutdown(ctx); err != nil {
				log.Fatal(err)
			}
			log.Println("server stopped")
			break Loop
		}
	}
}
