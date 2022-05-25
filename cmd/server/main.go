package main

import (
	"log"
	"net/http"

	"github.com/colzphml/yandex_project/internal/handlers"
)

func main() {
	http.HandleFunc("/status", handlers.StatusHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
