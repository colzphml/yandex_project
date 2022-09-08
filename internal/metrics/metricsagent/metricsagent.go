package metricsagent

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/colzphml/yandex_project/internal/agentutils"
	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/rs/zerolog"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

type MetricRepo struct {
	db map[string]metrics.Metrics
	mu sync.Mutex
}

func NewRepo() *MetricRepo {
	r := MetricRepo{
		db: make(map[string]metrics.Metrics),
	}
	return &r
}

var log = zerolog.New(agentutils.LogConfig()).With().Timestamp().Str("component", "metricsagent").Logger()

func GetRuntimeMetric(m *runtime.MemStats, fieldName string, fieldType string) (metrics.Metrics, error) {
	var result metrics.Metrics
	result.ID = fieldName
	result.MType = fieldType
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	f := r.FieldByName(fieldName)
	if f == (reflect.Value{}) {
		return metrics.Metrics{}, errors.New("runtime not have this variable:" + fieldName + ", check config file")
	}
	switch t := r.FieldByName(fieldName).Type().Name(); {
	case strings.Contains(t, "int") && fieldType == "gauge":
		var v float64
		v = float64(f.Uint())
		result.Value = &v
		return result, nil
	case strings.Contains(t, "int") && fieldType == "counter":
		var v int64
		v = int64(f.Uint())
		result.Delta = &v
		return result, nil
	case strings.Contains(t, "float") && fieldType == "gauge":
		var v float64
		v = float64(f.Float())
		result.Value = &v
		return result, nil
	case strings.Contains(t, "float") && fieldType == "counter":
		var v int64
		v = int64(f.Float())
		result.Delta = &v
		return result, nil
	default:
		return metrics.Metrics{}, errors.New("not know type of variable: " + fieldType + ", check config file")
	}
}

func ReadRuntimeMetrics(repo *MetricRepo, metricsDescr map[string]string, runtime *runtime.MemStats, inc int64) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for k, v := range metricsDescr {
		value, err := GetRuntimeMetric(runtime, k, v)
		if err != nil {
			log.Error().Err(err).Msg("failed collect metric")
			continue
		}
		repo.db[k] = value
	}
	incMetrics := metrics.Metrics{ID: "PollCount", MType: "counter", Delta: &inc}
	repo.db[incMetrics.ID] = incMetrics
	randomValue := rand.Float64()
	randMetrics := metrics.Metrics{ID: "RandomValue", MType: "gauge", Value: &randomValue}
	repo.db[randMetrics.ID] = randMetrics
}

func CollectRuntimeWorker(ctx context.Context, wg *sync.WaitGroup, cfg *agentutils.AgentConfig, repo *MetricRepo) {
	var runtimeState runtime.MemStats
	var pollCouter int64
	tickerPoll := time.NewTicker(cfg.PollInterval)
	for {
		select {
		case <-tickerPoll.C:
			runtime.ReadMemStats(&runtimeState)
			ReadRuntimeMetrics(repo, cfg.Metrics, &runtimeState, pollCouter)
			pollCouter++
		case <-ctx.Done():
			tickerPoll.Stop()
			log.Info().Msg("stopped collectWorker Runtime")
			wg.Done()
			return
		}
	}
}

func GetVirtualMemoryMetrics() ([]metrics.Metrics, error) {
	var result []metrics.Metrics
	vmem, err := mem.VirtualMemory()
	if err != nil {
		log.Error().Err(err).Msg("failed get virtual memory")
		return nil, err
	}
	result = append(result, GetTotalMemory(vmem))
	result = append(result, GetFreeMemory(vmem))
	return result, nil
}

func GetTotalMemory(vmem *mem.VirtualMemoryStat) metrics.Metrics {
	m := metrics.Metrics{}
	m.ID = "TotalMemory"
	m.MType = "gauge"
	valueTotal := float64(vmem.Total)
	m.Value = &valueTotal
	return m
}

func GetFreeMemory(vmem *mem.VirtualMemoryStat) metrics.Metrics {
	m := metrics.Metrics{}
	m.ID = "FreeMemory"
	m.MType = "gauge"
	valueFree := float64(vmem.Free)
	m.Value = &valueFree
	return m
}

func GetCPUMetrics() ([]metrics.Metrics, error) {
	var result []metrics.Metrics
	totalCPU, err := cpu.Counts(true)
	if err != nil {
		log.Error().Err(err).Msg("failed get total cpu count")
		return nil, err
	}
	CPUutil, err := cpu.Percent(0, true)
	if err != nil {
		log.Error().Err(err).Msg("failed get cpu util")
		return nil, err
	}
	for i := 1; i <= totalCPU; i++ {
		m := metrics.Metrics{}
		m.ID = "CPUutilization" + strconv.Itoa(i)
		m.MType = "gauge"
		value := CPUutil[i-1]
		m.Value = &value
		result = append(result, m)
	}
	return result, nil
}

func ReadSystemMetrics(repo *MetricRepo) {
	repo.mu.Lock()
	defer repo.mu.Unlock()
	virtualmemory, err := GetVirtualMemoryMetrics()
	if err != nil {
		log.Error().Err(err)
	} else {
		for _, v := range virtualmemory {
			repo.db[v.ID] = v
		}
	}
	cpumetrics, err := GetCPUMetrics()
	if err != nil {
		log.Error().Err(err)
		return
	}
	for _, v := range cpumetrics {
		repo.db[v.ID] = v
	}
}

func CollectSystemWorker(ctx context.Context, wg *sync.WaitGroup, cfg *agentutils.AgentConfig, repo *MetricRepo) {
	tickerPoll := time.NewTicker(cfg.PollInterval)
	for {
		select {
		case <-tickerPoll.C:
			ReadSystemMetrics(repo)
		case <-ctx.Done():
			tickerPoll.Stop()
			log.Info().Msg("stopped collectWorker System")
			wg.Done()
			return
		}
	}
}

func SendMetrics(srv string, repo *MetricRepo, client *http.Client) {
	var urlPrefix, urlPart string
	urlPrefix = "http://" + srv
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for k, v := range repo.db {
		urlPart = "/update/" + v.MType + "/" + k + "/" + v.ValueString()
		err := agentutils.HTTPSend(client, urlPrefix+urlPart)
		if err != nil {
			log.Error().Err(err).Msg("failed send metrics by url")
			continue
		}
	}
}

func SendJSONMetrics(srv string, key string, repo *MetricRepo, client *http.Client) {
	urlPrefix := "http://" + srv + "/update/"
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, v := range repo.db {
		v.FillHash(key)
		postBody, err := json.Marshal(v)
		if err != nil {
			log.Error().Err(err).Msg("failed marshall json")
			continue
		}
		err = agentutils.HTTPSendJSON(client, urlPrefix, postBody)
		if err != nil {
			log.Error().Err(err).Msg("failed send with body")
			continue
		}
	}
}

func SendListJSONMetrics(srv string, key string, repo *MetricRepo, client *http.Client) {
	urlPrefix := "http://" + srv + "/updates/"
	var list []metrics.Metrics
	repo.mu.Lock()
	defer repo.mu.Unlock()
	for _, v := range repo.db {
		v.FillHash(key)
		list = append(list, v)
	}
	postBody, err := json.Marshal(list)
	if err != nil {
		log.Error().Err(err).Msg("failed marshall json")
	}
	err = agentutils.HTTPSendJSON(client, urlPrefix, postBody)
	if err != nil {
		log.Error().Err(err).Msg("failed send with body (list)")
	}
}

func SendWorker(ctx context.Context, wg *sync.WaitGroup, cfg *agentutils.AgentConfig, repo *MetricRepo) {
	tickerReport := time.NewTicker(cfg.ReportInterval)
	client := &http.Client{}
	for {
		select {
		case <-tickerReport.C:
			SendListJSONMetrics(cfg.ServerAddress, cfg.Key, repo, client)
		case <-ctx.Done():
			tickerReport.Stop()
			log.Info().Msg("stopped sendWorker")
			wg.Done()
			return
		}
	}
}
