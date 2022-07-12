package main

import (
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/colzphml/yandex_project/internal/agentutils"
	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/metrics/metricsagent"
	"github.com/rs/zerolog"
)

var log = zerolog.New(agentutils.LogConfig()).With().Timestamp().Str("component", "agent").Logger()

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
	//testtest
	client := &http.Client{}
Loop:
	for {
		select {
		case <-tickerPoll.C:
			runtime.ReadMemStats(&runtimeState)
			metricsStore = metricsagent.CollectMetrics(cfg.Metrics, &runtimeState, pollCouter)
			pollCouter++
		case <-tickerReport.C:
			//send by url
			//metrics.SendMetrics(cfg.ServerAddress, metricsStore, client)
			//send by json
			//metrics.SendJSONMetrics(cfg.ServerAddress, cfg.Key, metricsStore, client)
			metricsagent.SendListJSONMetrics(cfg.ServerAddress, cfg.Key, metricsStore, client)
		case <-sigChan:
			tickerPoll.Stop()
			tickerReport.Stop()
			log.Info().Msg("initialize table")
			break Loop
		}
	}
}
