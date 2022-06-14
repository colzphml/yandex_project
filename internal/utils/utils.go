package utils

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/caarlos0/env"
	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	ServerAddress  string            `yaml:"ServerAddress" env:"ADDRESS"`
	PollInterval   time.Duration     `yaml:"PollInterval" env:"POLL_INTERVAL"`
	ReportInterval time.Duration     `yaml:"ReportInterval" env:"REPORT_INTERVAL"`
	Metrics        map[string]string `yaml:"Metrics"`
}

func LoadAgentConfig() *AgentConfig {
	//default config
	cfg := &AgentConfig{
		ServerAddress:  "127.0.0.1:8080",
		PollInterval:   time.Duration(2 * time.Second),
		ReportInterval: time.Duration(10 * time.Second),
		Metrics: map[string]string{
			"Alloc":         "gauge",
			"BuckHashSys":   "gauge",
			"Frees":         "gauge",
			"GCCPUFraction": "gauge",
			"GCSys":         "gauge",
			"HeapAlloc":     "gauge",
			"HeapIdle":      "gauge",
			"HeapInuse":     "gauge",
			"HeapObjects":   "gauge",
			"HeapReleased":  "gauge",
			"HeapSys":       "gauge",
			"LastGC":        "gauge",
			"Lookups":       "gauge",
			"MCacheInuse":   "gauge",
			"MCacheSys":     "gauge",
			"MSpanInuse":    "gauge",
			"MSpanSys":      "gauge",
			"Mallocs":       "gauge",
			"NextGC":        "gauge",
			"NumForcedGC":   "gauge",
			"NumGC":         "gauge",
			"OtherSys":      "gauge",
			"PauseTotalNs":  "gauge",
			"StackInuse":    "gauge",
			"StackSys":      "gauge",
			"Sys":           "gauge",
			"TotalAlloc":    "gauge",
		},
	}
	//yaml config
	yfile, err := ioutil.ReadFile("agent_config.yaml")
	if err != nil {
		log.Println(err.Error())
	} else {
		err = yaml.Unmarshal(yfile, &cfg)
		if err != nil {
			log.Println(err.Error())
		}
	}
	//env config
	err = env.Parse(cfg)
	if err != nil {
		log.Println(err.Error())
	}
	return cfg
}

type ServerConfig struct {
	ServerAddress string        `yaml:"ServerAddress" env:"ADDRESS"`
	StoreInterval time.Duration `yaml:"StoreInterval" env:"STORE_INTERVAL"`
	StoreFile     string        `yaml:"StoreFile" env:"STORE_FILE"`
	Restore       bool          `yaml:"Restore" env:"RESTORE"`
}

func LoadServerConfig() *ServerConfig {
	//default config
	cfg := &ServerConfig{
		ServerAddress: "127.0.0.1:8080",
		StoreInterval: time.Duration(300 * time.Second),
		StoreFile:     "/tmp/devops-metrics-db.json",
		Restore:       true,
	}
	//yaml config
	yfile, err := ioutil.ReadFile("server_config.yaml")
	if err != nil {
		log.Println(err.Error())
	} else {
		err = yaml.Unmarshal(yfile, &cfg)
		if err != nil {
			log.Println(err.Error())
		}
	}
	//env config
	err = env.Parse(cfg)
	if err != nil {
		log.Println(err.Error())
	}
	return cfg
}

func HTTPSend(client *http.Client, url string) error {
	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return err
	}
	return nil
}

func HTTPSendMetric(client *http.Client, url string, postBody []byte) error {
	body := bytes.NewBuffer(postBody)
	request, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return err
	}
	return nil
}
