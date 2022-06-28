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

	"github.com/colzphml/yandex_project/internal/agentutils"
	"github.com/colzphml/yandex_project/internal/metrics"
)

func main() {
	//read config file
	cfg := agentutils.LoadAgentConfig()
	//variables for collected data
	var runtimeState runtime.MemStats
	metricsStore := make(map[string]metrics.Metrics)
	//for close programm by signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	//for additional metric PollCount
	var pollCouter int64
	//for additional metric RandomValue
	rand.Seed(time.Now().UnixNano())
	//schedule ticker
	tickerPoll := time.NewTicker(cfg.PollInterval)
	tickerReport := time.NewTicker(cfg.ReportInterval)
	//client for send
	client := &http.Client{}
Loop:
	for {
		select {
		case <-tickerPoll.C:
			runtime.ReadMemStats(&runtimeState)
			metricsStore = metrics.CollectMetrics(cfg, &runtimeState, pollCouter)
			pollCouter++
		case <-tickerReport.C:
			//send by url
			//metrics.SendMetrics(cfg, metricsStore, client)
			//send by json
			metrics.SendJSONMetrics(cfg, metricsStore, client)
		case <-sigChan:
			tickerPoll.Stop()
			tickerReport.Stop()
			log.Println("close program")
			break Loop
		}
	}
}
