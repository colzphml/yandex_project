package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/colzphml/yandex_project/internal/handlers"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/colzphml/yandex_project/internal/utils"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := utils.LoadServerConfig()
	repo := storage.NewMetricRepo()
	repoJson := storage.NewMetricRepo()
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Post("/update/{metric_type}/{metric_name}/{metric_value}", handlers.SaveHandler(repo))
	r.Post("/update/", handlers.SaveJSONHandler(repoJson))
	r.Get("/value/{metric_type}/{metric_name}", handlers.GetValueHandler(repo))
	r.Post("/value/", handlers.GetJSONValueHandler(repoJson))
	r.Get("/", handlers.ListMetricsHandler(repo))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(cfg.ServerPort), r))
}
