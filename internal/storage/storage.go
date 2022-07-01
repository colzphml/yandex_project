package storage

import (
	"context"
	"time"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/colzphml/yandex_project/internal/storage/dbrepo"
	"github.com/colzphml/yandex_project/internal/storage/filerepo"
	"github.com/rs/zerolog"
)

var log = zerolog.New(serverutils.LogConfig()).With().Timestamp().Str("component", "storage").Logger()

type Repositorier interface {
	SaveMetric(ctx context.Context, metric metrics.Metrics) error
	SaveListMetric(ctx context.Context, metrics []metrics.Metrics) (int, error)
	ListMetrics(ctx context.Context) []string
	GetValue(ctx context.Context, metricName string) (metrics.Metrics, error)
	DumpMetrics(ctx context.Context, cfg *serverutils.ServerConfig) error
	Close()
	Ping(ctx context.Context) error
}

func CreateRepo(ctx context.Context, cfg *serverutils.ServerConfig) (Repositorier, *time.Ticker, error) {
	var tickerSave *time.Ticker
	tickerSave = &time.Ticker{}
	switch {
	//использование БД
	case cfg.DBDSN != "":
		repo, err := dbrepo.NewMetricRepo(ctx, cfg)
		if err != nil {
			return nil, nil, err
		}
		err = repo.Ping(ctx)
		if err != nil {
			return nil, nil, err
		}
		log.Info().Msg("used db")
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
		log.Info().Msg("used file")
		return repo, tickerSave, nil
	}
}
