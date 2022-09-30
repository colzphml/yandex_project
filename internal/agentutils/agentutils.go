// Модуль agentutils содержит в себе методы для работы агента, не зависящие от других модулей агента.
// Содержит в себе структуру для хранения конфигурации запуска агента и методы для чтения параметров запуска.
package agentutils

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

var log = zerolog.New(LogConfig()).With().Timestamp().Str("component", "agentutils").Logger()

// AgentConfig - конфигурация агента для старта.
type AgentConfig struct {
	Metrics        map[string]string `yaml:"Metrics"`                              // Описание метрик, собираемых из runtime
	Key            string            `yaml:"Key" env:"KEY"`                        // Ключ для подписи данных
	ServerAddress  string            `yaml:"ServerAddress" env:"ADDRESS"`          // Адрес сервера обработки метрик
	PollInterval   time.Duration     `yaml:"PollInterval" env:"POLL_INTERVAL"`     // Интервал сбора метрик агентом
	ReportInterval time.Duration     `yaml:"ReportInterval" env:"REPORT_INTERVAL"` // Интервал отправки данных на сервер
	PublicKey      *rsa.PublicKey    // Публичный ключ
}

// yamlRead - считывает yaml-файл конфигурации с названием "agent_config.yaml" и заполняет структуру AgentConfig.
func (cfg *AgentConfig) yamlRead(file string) {
	yfile, err := os.ReadFile(file)
	if err != nil {
		log.Error().Err(err).Msg("cannot open yaml file")
	} else {
		err = yaml.Unmarshal(yfile, &cfg)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse yaml file")
		}
	}
}

// envRead - считывает переменные окружения и заполняет структуру AgentConfig.
func (cfg *AgentConfig) envRead() {
	err := env.Parse(cfg)
	if err != nil {
		log.Error().Err(err).Msg("cannot read environment variables")
	}
	keypath := os.Getenv("CRYPTO_KEY")
	if keypath != "" {
		pk, err := getPublicKey(keypath)
		if err != nil {
			log.Error().Err(err).Msg("cannot get public key")
			return
		}
		cfg.PublicKey = pk
	}
}

// flagsRead - считывает флаги запуска и заполняет структуру AgentConfig.
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
	flag.Func("crypto-key", "path to public key", func(flagValue string) error {
		if flagValue != "" {
			pk, err := getPublicKey(flagValue)
			if err != nil {
				log.Error().Err(err).Msg("cannot get public key")
				return err
			}
			cfg.PublicKey = pk
		}
		return nil
	})
	flag.Parse()
}

// LoadAgentConfig - создает AgentConfig и заполняет его в следующем порядке:
//
// Значение по умолчанию -> YAML-файл -> переменные окружения -> флаги запуска.
//
// То, что находится правее в списке - будет в приоритете над тем, что левее.
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

// HTTPSend - производит POST запрос на указанный URL. В URL содержится вся необходимая информация (имя метрики, тип, значение)
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

// HTTPSendJSON - производит отправку json-метрики (в виде []byte) на сервер по указанному URL.
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

// LogConfig - настраивает формат логирования для zerolog.
func LogConfig() zerolog.ConsoleWriter {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	return output
}

func getPublicKey(file string) (*rsa.PublicKey, error) {
	byte, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(byte)
	if block == nil {
		return nil, errors.New("failed decode pem")
	}
	pk, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pk, nil
}
