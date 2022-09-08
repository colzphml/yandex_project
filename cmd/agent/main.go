package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/colzphml/yandex_project/internal/agentutils"
	"github.com/colzphml/yandex_project/internal/metrics/metricsagent"
	"github.com/rs/zerolog"
)

var log = zerolog.New(agentutils.LogConfig()).With().Timestamp().Str("component", "agent").Logger()

func startHTTPserver(wg *sync.WaitGroup) *http.Server {
	srv := &http.Server{Addr: ":8081"}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := srv.ListenAndServe(); errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err)
		}
	}()
	return srv
}

func main() {
	log.Info().Msg("agent started")
	now := time.Now()
	//read config file
	cfg := agentutils.LoadAgentConfig()
	wg := &sync.WaitGroup{}
	metricsStore := metricsagent.NewRepo()
	//for close programm by signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	//for additional metric RandomValue
	rand.Seed(time.Now().UnixNano())
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(3)
	go metricsagent.CollectRuntimeWorker(ctx, wg, cfg, metricsStore)
	go metricsagent.CollectSystemWorker(ctx, wg, cfg, metricsStore)
	go metricsagent.SendWorker(ctx, wg, cfg, metricsStore)
	srv := startHTTPserver(wg)
	<-sigChan
	srv.Shutdown(ctx)
	cancel()
	wg.Wait()
	log.Info().Msg(fmt.Sprintf("agent stopped after %v seconds of work", time.Since(now).Seconds()))
}
