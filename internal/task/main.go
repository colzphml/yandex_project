package main

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Adress         string        `env:"ADRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
}

func main() {
	cfg := &Config{
		Adress:         "127.0.0.1",
		ReportInterval: time.Duration(2 * time.Second),
		PollInterval:   time.Duration(10 * time.Second),
	}
	fmt.Println(cfg)
	err := env.Parse(cfg)
	if err != nil {
		panic(err)
	}
	fmt.Println(cfg)
}
