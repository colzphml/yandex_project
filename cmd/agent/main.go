package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/utils"
)

func main() {
	//read config file
	cfg := utils.LoadConfig()
	//variables for send data
	var runtimeState runtime.MemStats
	metricsStore := make(map[string]metrics.MetricValue)
	//for close programm by signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	//for additional metric PollCount
	var pollCouter metrics.Counter = 0
	//for additional metric RandomValue
	rand.Seed(time.Now().UnixNano())
	//can we get collision every 5th tickerPoll and every tickerReport??? Maybe send in other goroutine??
	tickerPoll := time.NewTicker(time.Duration(cfg.PollInterval) * time.Second)
	tickerReport := time.NewTicker(time.Duration(cfg.ReportInterval) * time.Second)
	//client for send
	client := &http.Client{}
	//maybe there is a better way
Loop:
	for {
		select {
		case <-tickerPoll.C:
			runtime.ReadMemStats(&runtimeState)
			metricsStore = metrics.CollectMetrics(cfg, &runtimeState, pollCouter)
			pollCouter++
		case <-tickerReport.C:
			metrics.SendMetrics(cfg, metricsStore, client)
		case <-sigChan:
			log.Println("close program")
			break Loop
		}
	}
}
