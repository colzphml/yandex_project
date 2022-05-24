package utils

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v3"
)

type AgentConfig struct {
	ServerAdress   string `yaml:"ServerAdress"`
	ServerPort     int    `yaml:"ServerPort"`
	PollInterval   int    `yaml:"PollInterval"`
	ReportInterval int    `yaml:"ReportInterval"`
}

func LoadConfig() *AgentConfig {
	cfg := &AgentConfig{
		ServerAdress:   "127.0.0.1",
		ServerPort:     8080,
		PollInterval:   2,
		ReportInterval: 10,
	}
	yfile, err := ioutil.ReadFile("agent_config.yaml")
	if err != nil {
		log.Println(err.Error())
		return cfg
	}
	err = yaml.Unmarshal(yfile, &cfg)
	if err != nil {
		log.Println(err.Error())
		return cfg
	}
	return cfg
}
