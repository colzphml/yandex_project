package storage

import (
	"log"
	"time"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/colzphml/yandex_project/internal/storage/dbrepo"
	"github.com/colzphml/yandex_project/internal/storage/filerepo"
)

type Repositorier interface {
	SaveMetric(metric metrics.Metrics) error
	ListMetrics() []string
	GetValue(metricName string) (metrics.Metrics, error)
	DumpMetrics(cfg *serverutils.ServerConfig) error
	Close()
	Ping() error
}

func CreateRepo(cfg *serverutils.ServerConfig) (Repositorier, *time.Ticker, error) {
	var tickerSave *time.Ticker
	tickerSave = &time.Ticker{}
	switch {
	//использование БД
	case cfg.DBDSN != "":
		repo, err := dbrepo.NewMetricRepo(cfg)
		if err != nil {
			return nil, nil, err
		}
		err = repo.Ping()
		if err != nil {
			return nil, nil, err
		}
		log.Println("used db")
		return repo, tickerSave, nil
	//использование файла
	case cfg.StoreFile != "":
		repo, err := filerepo.NewMetricRepo(cfg)
		if err != nil {
			return nil, nil, err
		}
		if cfg.StoreInterval.Seconds() != 0 {
			tickerSave = time.NewTicker(cfg.StoreInterval)
		}
		return repo, tickerSave, nil
	default:
		repo, err := filerepo.NewMetricRepo(cfg)
		if err != nil {
			return nil, nil, err
		}
		log.Println("used file")
		return repo, tickerSave, nil
	}
}
