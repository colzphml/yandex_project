package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/colzphml/yandex_project/internal/handlers"
	"github.com/colzphml/yandex_project/internal/utils"
)

func main() {
	cfg := utils.LoadServerConfig()
	http.HandleFunc("/update/", handlers.StatusHandler)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(cfg.ServerPort), nil))
}
