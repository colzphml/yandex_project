package main

import (
	"context"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/colzphml/yandex_project/internal/app/server"
	"github.com/colzphml/yandex_project/internal/app/server/serverutils"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/rs/zerolog"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
	log                 = zerolog.New(serverutils.LogConfig()).With().Timestamp().Str("component", "server").Logger()
)

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
	srv := server.HTTPServer(ctx, cfg, repo)
	grpcsrv := server.GRPCServer(ctx, cfg, repo)
	wg := &sync.WaitGroup{}
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
			wg.Add(1)
			go func() {
				grpcsrv.GracefulStop()
				log.Info().Msg("grpc stopped")
				wg.Done()
			}()
			if err := srv.Shutdown(ctxcancel); err != nil {
				log.Error().Err(err).Msg("failed shutdown server")
			}
			wg.Wait()
			log.Info().Msg("server stopped")
			break Loop
		}
	}
}
