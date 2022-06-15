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
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/colzphml/yandex_project/internal/utils_server"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func HTTPServer(cfg *utils_server.ServerConfig, repo *storage.MetricRepo, repoJSON *storage.MetricRepo) *http.Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/update/{metric_type}/{metric_name}/{metric_value}", handlers.SaveHandler(repo))
	r.Post("/update/", handlers.SaveJSONHandler(repoJSON, cfg))
	r.Get("/value/{metric_type}/{metric_name}", handlers.GetValueHandler(repo))
	r.Post("/value/", handlers.GetJSONValueHandler(repoJSON))
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
	cfg := utils_server.LoadServerConfig()
	repo, err := storage.NewMetricRepo(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	repoJSON, err := storage.NewMetricRepo(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	if cfg.StoreInterval != 0*time.Second {
		tickerSave := time.NewTicker(cfg.StoreInterval)
		srv := HTTPServer(cfg, repo, repoJSON)
	Loop:
		for {
			select {
			case <-tickerSave.C:
				err := repoJSON.StoreMetric(cfg)
				if err != nil {
					log.Println(err.Error())
				}
			case <-sigChan:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer func() {
					repoJSON.StoreMetric(cfg)
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
	} else {
		srv := HTTPServer(cfg, repo, repoJSON)
		<-sigChan
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer func() {
			repoJSON.StoreMetric(cfg)
			cancel()
		}()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal(err)
		}
		log.Println("server stopped")
	}
}
