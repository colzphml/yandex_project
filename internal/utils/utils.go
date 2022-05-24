package utils

import (
	"io/ioutil"
	"log"
	"net/http"

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
