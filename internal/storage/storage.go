package storage

import (
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
	case cfg.DbDSN != "":
		repo, err := dbrepo.NewMetricRepo(cfg)
		if err != nil {
			return nil, err
		}
		return repo, nil
	default:
		repo, err := filerepo.NewMetricRepo(cfg)
		if err != nil {
			return nil, err
		}
		return repo, nil
	}
}
