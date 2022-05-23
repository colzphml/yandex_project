package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/colzphml/yandex_project/internal/metrics"
	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	ServerAdress   string
	ServerPort     int
	PollInterval   int
	ReportInterval int
}

func NewAgentConfig() AgentConfig {
	agentConfig := AgentConfig{}
	agentConfig.PollInterval = 2
	agentConfig.ReportInterval = 10
	agentConfig.ServerAdress = "127.0.0.1"
	agentConfig.ServerPort = 8080
	return agentConfig
}

func LoadConfig() AgentConfig {
	cfg := NewAgentConfig()
	yfile, err := ioutil.ReadFile("agent_config.yaml")
	if err != nil {
		log.Println(err.Error())
		return cfg
	}
	data := make(map[string]interface{})
	err = yaml.Unmarshal(yfile, &data)
	if err != nil {
		log.Println(err.Error())
		return cfg
	}
	if val, ok := data["pollInterval"]; ok {
		cfg.PollInterval = val.(int)
	}
	if val, ok := data["reportInterval"]; ok {
		cfg.ReportInterval = val.(int)
	}
	if val, ok := data["serverPort"]; ok {
		cfg.ServerPort = val.(int)
	}
	if val, ok := data["serverAdress"]; ok {
		cfg.ServerAdress = val.(string)
	}
	return cfg
}

func getRuntimeMetric(m runtime.MemStats, field string) metrics.Gauge {
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	f := r.FieldByName(field)
	switch t := r.FieldByName(field).Type().Name(); {
	case strings.Contains(t, "int"):
		return metrics.Gauge(f.Uint())
	case strings.Contains(t, "float"):
		return metrics.Gauge(f.Float())
	default:
		log.Printf("unknown type of metric: %v", r.FieldByName(field).Type().Name())
	}
	return metrics.Gauge(f.Float())
}

func SetMetrics(runtime *metrics.RuntimeMetrics, addit *metrics.AdditionalMetrics, runtimeState runtime.MemStats, inc metrics.Counter) {
	r := reflect.ValueOf(runtime)
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	for j := 0; j < r.NumField(); j++ {
		switch r.Field(j).Type().Name() {
		case "Gauge":
			r.Field(j).SetFloat(float64(getRuntimeMetric(runtimeState, r.Type().Field(j).Name)))
		case "Counter":
			r.Field(j).SetInt(int64(getRuntimeMetric(runtimeState, r.Type().Field(j).Name)))
		default:
			log.Println("undefined type for get")
		}
	}
	addit.PollCount = inc
	addit.RandomValue = metrics.Gauge(rand.Float64())
}

func SendMetrics(cfg AgentConfig, input interface{}, client http.Client) {
	var r reflect.Value
	switch input := input.(type) {
	case metrics.RuntimeMetrics:
		metrics := input
		r = reflect.ValueOf(&metrics)
	case metrics.AdditionalMetrics:
		metrics := input
		r = reflect.ValueOf(&metrics)
	}
	urlPrefix := fmt.Sprintf("http://%v:%v/update", cfg.ServerAdress, cfg.ServerPort)
	urlPart := ""
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	for j := 0; j < r.NumField(); j++ {
		switch r.Field(j).Type().Name() {
		case "Gauge":
			urlPart = fmt.Sprintf("/%v/%v/%v", r.Field(j).Type().Name(), r.Type().Field(j).Name, metrics.Gauge(r.Field(j).Float()))
		case "Counter":
			urlPart = fmt.Sprintf("/%v/%v/%v", r.Field(j).Type().Name(), r.Type().Field(j).Name, metrics.Counter(r.Field(j).Int()))
		default:
			log.Println("undefined type for send")
			continue
		}
		request, err := http.NewRequest(http.MethodPost, urlPrefix+urlPart, nil)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		request.Header.Set("Content-Type", "text/plain")
		response, err := client.Do(request)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		if response.StatusCode != 200 {
			log.Println("send failed")
			continue
		}
	}
}
