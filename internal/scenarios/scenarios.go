// Package scenarios описывает логику работы с данными.
package scenarios

import (
	"context"
	"errors"
	"fmt"

	"github.com/colzphml/yandex_project/internal/app/server/serverutils"
	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/rs/zerolog/log"
)

// Ошибки при работе с данными
var (
	ErrStatusNotFound            = errors.New("status not found (404)")
	ErrStatusBadRequest          = errors.New("wrong request (400)")
	ErrStatusNotImplemented      = errors.New("wrong type (501)")
	ErrStatusInternalServerError = errors.New("internal server error(500)")
)

func SaveMetric(ctx context.Context, repo storage.Repositorier, cfg *serverutils.ServerConfig, metric metrics.Metrics, sign bool) error {
	if sign {
		compareHash, err := metric.CompareHash(cfg.Key)
		if err != nil {
			return ErrStatusInternalServerError
		}
		if !compareHash {
			return fmt.Errorf("signature is wrong: %w", ErrStatusBadRequest)
		}
	}
	err := repo.SaveMetric(ctx, metric)
	if err != nil {
		return ErrStatusBadRequest
	}
	if cfg.StoreInterval.Nanoseconds() == 0 {
		err = repo.DumpMetrics(ctx, cfg)
		if err != nil {
			return ErrStatusInternalServerError
		}
	}
	return nil
}

func SaveArrayMetric(ctx context.Context, repo storage.Repositorier, cfg *serverutils.ServerConfig, metrics []metrics.Metrics) (int, error) {
	for _, v := range metrics {
		compareHash, err := v.CompareHash(cfg.Key)
		if err != nil {
			return 0, ErrStatusInternalServerError
		}
		if !compareHash {
			return 0, ErrStatusBadRequest
		}
	}
	count, err := repo.SaveListMetric(ctx, metrics)
	if err != nil {
		log.Error().Err(err).Msg("can't save metric")
		return 0, ErrStatusBadRequest
	}
	if cfg.StoreInterval.Nanoseconds() == 0 {
		err = repo.DumpMetrics(ctx, cfg)
		if err != nil {
			return 0, ErrStatusInternalServerError
		}
	}
	return count, nil
}

func GetMetric(ctx context.Context, repo storage.Repositorier, cfg *serverutils.ServerConfig, name string, mtype string, sign bool) (metrics.Metrics, error) {
	metricValue, err := repo.GetValue(ctx, name)
	if err != nil {
		return metrics.Metrics{}, ErrStatusNotFound
	}
	if metricValue.MType != mtype {
		return metrics.Metrics{}, fmt.Errorf("this metric have another type: %w", ErrStatusNotFound)
	}
	if sign {
		err = metricValue.FillHash(cfg.Key)
		if err != nil {

			return metrics.Metrics{}, ErrStatusInternalServerError
		}
	}
	return metricValue, nil
}
