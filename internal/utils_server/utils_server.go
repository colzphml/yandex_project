package utils_server

import (
	"flag"
	"io/ioutil"
	"log"
	"time"

	"github.com/caarlos0/env"
	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	ServerAddress string        `yaml:"ServerAddress" env:"ADDRESS"`
	StoreInterval time.Duration `yaml:"StoreInterval" env:"STORE_INTERVAL"`
	StoreFile     string        `yaml:"StoreFile" env:"STORE_FILE"`
	Restore       bool          `yaml:"Restore" env:"RESTORE"`
}

func (cfg *ServerConfig) yamlRead(file string) {
	yfile, err := ioutil.ReadFile(file)
	if err != nil {
		log.Println(err.Error())
	} else {
		err = yaml.Unmarshal(yfile, &cfg)
		if err != nil {
			log.Println(err.Error())
		}
	}
}

func (cfg *ServerConfig) envRead() {
	err := env.Parse(cfg)
	if err != nil {
		log.Println(err.Error())
	}
}

func (cfg *ServerConfig) flagsRead() {
	flag.StringVar(&cfg.ServerAddress, "a", "127.0.0.1:8080", "server address like <server>:<port>")
	flag.StringVar(&cfg.StoreFile, "f", "./tmp/devops-metrics-db.json", "file for store metrics, example /root/myfile.json")
	flag.BoolVar(&cfg.Restore, "r", true, "true/false for restore metrics from disk after restart") //работает только если -r=false, если -r false нет. Почему?
	flag.DurationVar(&cfg.StoreInterval, "i", 300*time.Second, "time duration for store metrics, for example 20s")
	flag.Parse()
}

func LoadServerConfig() *ServerConfig {
	//default config
	cfg := &ServerConfig{
		ServerAddress: "127.0.0.1:8080",
		StoreInterval: time.Duration(300 * time.Second),
		StoreFile:     "./tmp/devops-metrics-db.json",
		Restore:       true,
	}
	//yaml config
	cfg.yamlRead("server_config.yaml")
	//flags config
	cfg.flagsRead()
	//env config
	cfg.envRead()
	return cfg
}
