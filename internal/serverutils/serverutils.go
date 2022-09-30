// Модуль serverutils содержит в себе методы для работы сервера, не зависящие от других модулей сервера.
// Содержит в себе структуру для хранения конфигурации запуска сервера и методы для чтения параметров запуска.
package serverutils

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v3"
)

var log = zerolog.New(LogConfig()).With().Timestamp().Str("component", "serverutils").Logger()

// ServerConfig - конфигурация сервера для старта.
type ServerConfig struct {
	DBDSN         string          `yaml:"DBDSN" env:"DATABASE_DSN"`           // URL для подключения к Postgres
	Key           string          `yaml:"Key" env:"KEY"`                      // Ключ для подписи данных
	ServerAddress string          `yaml:"ServerAddress" env:"ADDRESS"`        // Адрес, по которому будут доступны endpoints
	StoreFile     string          `yaml:"StoreFile" env:"STORE_FILE"`         // Адрес файла для хранения метрик
	Restore       bool            `yaml:"Restore" env:"RESTORE"`              // При true - значения метрик в памяти сервера восстановится из хранилища, при false - в памяти будет пустое хранилище
	StoreInterval time.Duration   `yaml:"StoreInterval" env:"STORE_INTERVAL"` // Интервал сохраниения данных при использовании файла как хранилища
	PrivateKey    *rsa.PrivateKey // приватный ключ
}

// yamlRead - считывает yaml-файл конфигурации с названием "server_config.yaml" и заполняет структуру ServerConfig.
func (cfg *ServerConfig) yamlRead(file string) {
	yfile, err := os.ReadFile(file)
	if err != nil {
		log.Error().Err(err).Msg("file open trouble")
	} else {
		err = yaml.Unmarshal(yfile, &cfg)
		if err != nil {
			log.Error().Err(err).Msg("parse yaml err")
		}
	}
}

// envRead - считывает переменные окружения и заполняет структуру ServerConfig.
func (cfg *ServerConfig) envRead() {
	err := env.Parse(cfg)
	if err != nil {
		log.Error().Err(err).Msg("problem with environment read")
	}
	keypath := os.Getenv("CRYPTO_KEY")
	if keypath != "" {
		pk, err := getPrivateKey(keypath)
		if err != nil {
			log.Error().Err(err).Msg("cannot get private key")
			return
		}
		cfg.PrivateKey = pk
	}
}

// flagsRead - считывает флаги запуска и заполняет структуру ServerConfig.
func (cfg *ServerConfig) flagsRead() {
	flag.Func("a", "server address like <server>:<port>, example: -a \"127.0.0.1:8080\"", func(flagValue string) error {
		if flagValue != "" {
			cfg.ServerAddress = flagValue
		}
		return nil
	})
	flag.Func("f", "file for store metrics, example: -f \"/root/myfile.json\"", func(flagValue string) error {
		if flagValue != "" {
			cfg.StoreFile = flagValue
		}
		return nil
	})
	flag.Func("r", "true/false for restore metrics from disk after restart, example: -r=true", func(flagValue string) error {
		if flagValue != "" {
			value, err := strconv.ParseBool(flagValue)
			if err != nil {
				return err
			}
			cfg.Restore = value
		}
		return nil
	})
	flag.Func("i", "time duration for store metrics, example: -r \"100s\"", func(flagValue string) error {
		if flagValue != "" {
			interval, err := time.ParseDuration(flagValue)
			if err != nil {
				return err
			}
			cfg.StoreInterval = interval
		}
		return nil
	})
	flag.Func("k", "key for data hash, example: -k \"sample key\"", func(flagValue string) error {
		if flagValue != "" {
			cfg.Key = flagValue
		}
		return nil
	})
	flag.Func("d", "key for database DSN, example: -d \"postgres://pi:toor@192.168.1.69:5432/yandex\"", func(flagValue string) error {
		if flagValue != "" {
			cfg.DBDSN = flagValue
		}
		return nil
	})
	flag.Func("crypto-key", "path to private key", func(flagValue string) error {
		if flagValue != "" {
			pk, err := getPrivateKey(flagValue)
			if err != nil {
				log.Error().Err(err).Msg("cannot get private key")
				return err
			}
			cfg.PrivateKey = pk
		}
		return nil
	})
	flag.Parse()
}

// LoadServerConfig - создает ServerConfig и заполняет его в следующем порядке:
//
// Значение по умолчанию -> YAML-файл -> переменные окружения -> флаги запуска.
//
// То, что находится правее в списке - будет в приоритете над тем, что левее.
func LoadServerConfig() *ServerConfig {
	//default config
	cfg := &ServerConfig{
		ServerAddress: "127.0.0.1:8080",
		StoreInterval: time.Duration(300 * time.Second),
		StoreFile:     "./tmp/devops-metrics-db.json",
		Restore:       false,
		Key:           "",
	}
	//yaml config
	cfg.yamlRead("server_config.yaml")
	//flags config
	cfg.flagsRead()
	//env config
	cfg.envRead()
	return cfg
}

// LogConfig - настраивает формат логирования для zerolog.
func LogConfig() zerolog.ConsoleWriter {
	output := zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("| %-6s|", i))
	}
	return output
}

func getPrivateKey(file string) (*rsa.PrivateKey, error) {
	byte, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(byte)
	if block == nil {
		return nil, errors.New("failed decode pem")
	}
	pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return pk, nil
}
