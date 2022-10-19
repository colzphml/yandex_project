// Package agentutils содержит в себе методы для работы агента, не зависящие от других модулей агента.
// Содержит в себе структуру для хранения конфигурации запуска агента и методы для чтения параметров запуска.
package agentutils

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caarlos0/env"
	"github.com/rs/zerolog"
)

var log = zerolog.New(LogConfig()).With().Timestamp().Str("component", "agentutils").Logger()

// AgentConfig - конфигурация агента для старта.
type AgentConfig struct {
	Metrics        map[string]string // Описание метрик, собираемых из runtime
	Key            string            `env:"KEY"`                    // Ключ для подписи данных
	ServerAddress  string            `env:"ADDRESS" json:"address"` // Адрес сервера обработки метрик
	ConfigFile     string            `env:"CONFIG"`                 // Адрес файла конфигурации в формате JSON
	PollInterval   time.Duration     `env:"POLL_INTERVAL"`          // Интервал сбора метрик агентом
	ReportInterval time.Duration     `env:"REPORT_INTERVAL"`        // Интервал отправки данных на сервер
	PublicKey      *rsa.PublicKey    // Публичный ключ
}

func (cfg *AgentConfig) UnmarshalJSON(data []byte) error {
	// Этот тип вводится для того, что бы не получить рекурсию в вызове анмаршаллера.
	// Если укажем в качестве типа родной AgentConfig, так как для него указана функция JSONUnmarshal, при обработке элемента структуры будет снова вызываться эта же функция и так далее.
	// Это описано в уроках на курсе, где разбирается работа с JSON
	type AgentConfigAlias AgentConfig
	AliasValue := &struct {
		*AgentConfigAlias
		PublicKey      string `json:"crypto_key"`
		PollInterval   string `json:"poll_interval"`
		ReportInterval string `json:"report_interval"`
	}{
		AgentConfigAlias: (*AgentConfigAlias)(cfg),
	}
	if err := json.Unmarshal(data, AliasValue); err != nil {
		return err
	}
	if AliasValue.PublicKey != "" {
		pk, err := getPublicKey(AliasValue.PublicKey)
		if err != nil {
			log.Error().Err(err).Msg("cannot get private key")
			return err
		}
		cfg.PublicKey = pk
	}
	if AliasValue.PollInterval != "" {
		dur, err := time.ParseDuration(AliasValue.PollInterval)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse time duration")
			return err
		}
		cfg.PollInterval = dur
	}
	if AliasValue.ReportInterval != "" {
		// Да, можно было сделать тип type Duration time.Duration, но мне не нравится такой подход.
		// Изначально у меня вообще были типы type Gauge float64 и type Counter int64, но при работе с ними было много неудобств, так как они не реализуются большинство стандартных интерфейсов
		// Поэтому я от этой идеи отказался и в том числе здесь я просто описал как парсить этот тип в переопределенном анмаршаллере
		dur, err := time.ParseDuration(AliasValue.ReportInterval)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse time duration")
			return err
		}
		cfg.ReportInterval = dur
	}
	return nil
}

// jsonRead - считывает JSON-файл конфигурации с названием из параметра c/config или переменной окружения CONFIG и заполняет структуру AgentConfig.
func (cfg *AgentConfig) jsonRead(file string) {
	jfile, err := os.ReadFile(file)
	if err != nil {
		log.Error().Err(err).Msg("file open trouble")
		return
	}
	err = json.Unmarshal(jfile, &cfg)
	if err != nil {
		log.Error().Err(err).Msg("parse json err")
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
	flag.Func("c", "config JSON file path, example: -f \"/cfg.json\"", func(flagValue string) error {
		if flagValue != "" {
			cfg.ConfigFile = flagValue
		}
		return nil
	})
	flag.Func("config", "config JSON file path, example: -f \"/cfg.json\"", func(flagValue string) error {
		if flagValue != "" {
			cfg.ConfigFile = flagValue
		}
		return nil
	})
	flag.Parse()
}

// LoadAgentConfig - создает AgentConfig и заполняет его в следующем порядке:
//
// Значение по умолчанию -> JSON-файл -> переменные окружения -> флаги запуска.
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
	//flags config
	cfg.flagsRead()
	//env config
	cfg.envRead()
	if cfg.ConfigFile != "" {
		//json config
		cfg.jsonRead(cfg.ConfigFile)
		flag.Parse()
		cfg.envRead()
	}
	return cfg
}

// HTTPSend - производит POST запрос на указанный URL. В URL содержится вся необходимая информация (имя метрики, тип, значение)
func HTTPSend(client *http.Client, url string) error {
	request, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "text/plain")
	request.Header.Set("X-Real-IP", GetLocalIP())
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
	request.Header.Set("X-Real-IP", GetLocalIP())
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

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
