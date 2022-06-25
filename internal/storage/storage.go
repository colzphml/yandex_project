package storage

import (
	"log"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/colzphml/yandex_project/internal/storage/dbrepo"
	"github.com/colzphml/yandex_project/internal/storage/filerepo"
)

type Repositorier interface {
	SaveMetric(metric metrics.Metrics) error
	ListMetrics() []string
	GetValue(metricName string) (metrics.Metrics, error)
	StoreMetric(cfg *serverutils.ServerConfig) error
	Close()
	Ping() error
}

func CreateRepo(cfg *serverutils.ServerConfig) (Repositorier, error) {
	switch {
	case cfg.DBDSN != "":
		var repo Repositorier
		repo, err := dbrepo.NewMetricRepo(cfg)
		if err != nil {
			repo, err = filerepo.NewMetricRepo(cfg)
			log.Println("used file")
			if err != nil {
				return nil, err
			}
		} else {
			log.Println("used db")
		}
		err = repo.Ping()
		if err != nil {
			return nil, err
		}
		return repo, nil
	default:
		repo, err := filerepo.NewMetricRepo(cfg)
		if err != nil {
			return nil, err
		}
		log.Println("used file")
		return repo, nil
	}
}
