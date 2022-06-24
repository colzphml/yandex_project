package dbrepo

import (
	"context"
	"errors"
	"sort"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrNoDB = errors.New("connection to db was not created")
)

type MetricRepo struct {
	DB   map[string]metrics.Metrics
	Pool *pgxpool.Pool
}

func NewMetricRepo(cfg *serverutils.ServerConfig) (*MetricRepo, error) {
	repo := &MetricRepo{
		DB: make(map[string]metrics.Metrics),
	}
	dbpool, err := pgxpool.Connect(context.Background(), cfg.DBDSN)
	if err != nil {
		return nil, err
	}
	repo.Pool = dbpool
	return repo, nil
}

func (m *MetricRepo) StoreMetric(cfg *serverutils.ServerConfig) error {
	/*
		if cfg.StoreFile != "" {
			file, err := os.OpenFile(cfg.StoreFile, os.O_RDWR|os.O_CREATE, 0777)
			if err != nil {
				return err
			}
			defer file.Close()
			return json.NewEncoder(file).Encode(m)
		}
	*/
	return nil
}

func (m *MetricRepo) Close() {
	m.Pool.Close()
}

func (m *MetricRepo) Ping() error {
	return m.Pool.Ping(context.Background())
}

func (m *MetricRepo) SaveMetric(metric metrics.Metrics) error {
	if v, ok := m.DB[metric.ID]; ok {
		newValue, err := metrics.NewValue(v, metric)
		if err != nil {
			return err
		}
		m.DB[metric.ID] = newValue
	} else {
		m.DB[metric.ID] = metric
	}
	return nil
}

func (m *MetricRepo) ListMetrics() []string {
	var list []string
	for k, v := range m.DB {
		list = append(list, k+":"+v.ValueString())
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i] < list[j]
	})
	return list
}

func (m *MetricRepo) GetValue(metricName string) (metrics.Metrics, error) {
	v, ok := m.DB[metricName]
	if !ok {
		return metrics.Metrics{}, errors.New("metric not saved")
	}
	return v, nil
}
