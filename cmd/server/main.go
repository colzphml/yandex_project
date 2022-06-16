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
func HTTPServer(cfg *serverutils.ServerConfig, repo *storage.MetricRepo, repoJSON *storage.MetricRepo) *http.Server {
	r := chi.NewRouter()
	r.Use(mdw.GzipHandle)
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
	//запуск в отдельной го рутине, что бы можно было повесить таймер на сохранение. Есть ли еще другие варианты реализации сохранения в storage по таймеру?
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
	repo, err := storage.NewMetricRepo(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	repoJSON, err := storage.NewMetricRepo(cfg)
	if err != nil {
		log.Fatal(err.Error())
	}
	//для "штатного" завершения сервера
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	/*
		Мне не очень нравится следующий кусок кода, но он нужен, если таймер сохранения равен нулю.
		Можно ли использовать такую конструкцию для "неработающего тикера", который никогда не сработает, но не надо будет дублировать код?
		var ticker *time.Ticker
		ticker = &time.Ticker{}
	*/
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
				//ура, контекст пригодился!
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer func() {
					repoJSON.StoreMetric(cfg)
					log.Println("metrics stored")
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
