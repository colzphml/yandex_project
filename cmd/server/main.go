package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/colzphml/yandex_project/internal/handlers"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/colzphml/yandex_project/internal/utils"
)

func main() {
	cfg := utils.LoadServerConfig()
	repo := storage.NewMetricRepo()
	http.HandleFunc("/update/", handlers.SaveHandler(&repo))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(cfg.ServerPort), nil))
}
