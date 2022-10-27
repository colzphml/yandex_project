// Модуль storage описывает хранилище данных по метрикам и отвечает за создание репозитория.
package storage

import (
	"context"
	"time"

	"github.com/colzphml/yandex_project/internal/app/server/serverutils"
	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/storage/dbrepo"
	"github.com/colzphml/yandex_project/internal/storage/filerepo"
	"github.com/rs/zerolog"
)

var log = zerolog.New(serverutils.LogConfig()).With().Timestamp().Str("component", "storage").Logger()

// Repositorier - интерфейс, описывающий работу с хранилищем метрик.
type Repositorier interface {
	SaveMetric(ctx context.Context, metric metrics.Metrics) error               // Сохранение отдельной метрики
	SaveListMetric(ctx context.Context, metrics []metrics.Metrics) (int, error) // Сохранение массива метрик
	ListMetrics(ctx context.Context) []metrics.Metrics                          // Получение списка метрик и их значений
	GetValue(ctx context.Context, metricName string) (metrics.Metrics, error)   // Получает метрику по ее имени из хранилища
	DumpMetrics(ctx context.Context, cfg *serverutils.ServerConfig) error       // Сохранение метрик из локальной памяти
	Close()                                                                     // Закрытие хранилища
	Ping(ctx context.Context) error                                             // Проверка доступности хранилища
}

// CreateRepo - создает хранилище на основе параметров сервера.
//
// Если указан URL Postgres - используется ДБ.
//
// Если указан файл, но не указан URL Postgres - используется файл.
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
