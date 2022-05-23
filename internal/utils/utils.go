package utils

import (
	"io/ioutil"
	"log"

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
