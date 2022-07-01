package agentutils

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

var log = zerolog.New(LogConfig()).With().Timestamp().Str("component", "agentutils").Logger()

type AgentConfig struct {
	ServerAddress  string            `yaml:"ServerAddress" env:"ADDRESS"`
	PollInterval   time.Duration     `yaml:"PollInterval" env:"POLL_INTERVAL"`
	ReportInterval time.Duration     `yaml:"ReportInterval" env:"REPORT_INTERVAL"`
	Key            string            `yaml:"Key" env:"KEY"`
	Metrics        map[string]string `yaml:"Metrics"`
}

func (cfg *AgentConfig) yamlRead(file string) {
	yfile, err := ioutil.ReadFile(file)
	if err != nil {
		log.Error().Err(err).Msg("cannot open yaml file")
	} else {
		err = yaml.Unmarshal(yfile, &cfg)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse yaml file")
		}
	}
}

func (cfg *AgentConfig) envRead() {
	err := env.Parse(cfg)
	if err != nil {
		log.Error().Err(err).Msg("cannot read eenvironment variables")
	}
}

func (cfg *AgentConfig) flagsRead() {
	flag.Func("a", "server address like <server>:<port>, example: -a \"127.0.0.1:8080\"", func(flagValue string) error {
		if flagValue != "" {
			cfg.ServerAddress = flagValue
		}
		return nil
	})
	flag.Func("r", "duration for send metrics to server, example: -r \"100s\"", func(flagValue string) error {
		if flagValue != "" {
			interval, err := time.ParseDuration(flagValue)
			if err != nil {
				return err
			}
			cfg.ReportInterval = interval
		}
		return nil
	})
	flag.Func("p", "duration for collect metrics to server, example: -p \"20s\"", func(flagValue string) error {
		if flagValue != "" {
			interval, err := time.ParseDuration(flagValue)
			if err != nil {
				return err
			}
			cfg.PollInterval = interval
		}
		return nil
	})
	flag.Func("k", "key for data hash, example: -k \"sample key\"", func(flagValue string) error {
		if flagValue != "" {
			cfg.Key = flagValue
		}
		return nil
	})
	flag.Parse()
}

func LoadAgentConfig() *AgentConfig {
	//default config
	cfg := &AgentConfig{
		ServerAddress:  "127.0.0.1:8080",
		PollInterval:   time.Duration(2 * time.Second),
		ReportInterval: time.Duration(10 * time.Second),
		Key:            "",
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
	cfg.yamlRead("agent_config.yaml")
	//flags config
	cfg.flagsRead()
	//env config
	cfg.envRead()
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
	return nil
}

func HTTPSendJSON(client *http.Client, url string, postBody []byte) error {
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

func LogConfig() zerolog.ConsoleWriter {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	return output
}
