package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/colzphml/yandex_project/internal/handlers"
	"github.com/colzphml/yandex_project/internal/utils"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

func main() {
	cfg := utils.LoadServerConfig()
	//repo := storage.NewMetricRepo()
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Get("/update/{metric_type}/{metric_name}/{metric_value}", handlers.TestFunc)

	//http.HandleFunc("/update/", handlers.SaveHandler(&repo))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(cfg.ServerPort), r))
}
