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
func HTTPServer(cfg *serverutils.ServerConfig, repo storage.Repositorier, repoJSON storage.Repositorier) *http.Server {
	r := chi.NewRouter()
	r.Use(mdw.GzipHandle)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/update/{metric_type}/{metric_name}/{metric_value}", handlers.SaveHandler(repo))
	r.Get("/value/{metric_type}/{metric_name}", handlers.GetValueHandler(repo))
	r.Post("/update/", handlers.SaveJSONHandler(repoJSON, cfg))
	r.Post("/value/", handlers.GetJSONValueHandler(repoJSON, cfg))
	r.Get("/ping", handlers.PingHandler(repoJSON, cfg))
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
	//сделал отдельный сторадж для запросов по url и json
	repo, err := storage.CreateRepo(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	repoJSON, err := storage.CreateRepo(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	//для "штатного" завершения сервера
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	var tickerSave *time.Ticker
	if cfg.StoreInterval.Seconds() != 0 {
		tickerSave = time.NewTicker(cfg.StoreInterval)
	}
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
				repo.StoreMetric(cfg)
				log.Println("metrics stored")
				repo.Close()
				repoJSON.Close()
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
