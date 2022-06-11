package storage

import (
	"errors"
	"sort"

	"github.com/colzphml/yandex_project/internal/metrics"
)

type Repositorier interface {
	SaveMetric(metric metrics.Metrics) error
	ListMetrics() []string
	GetValue(metricName string) (metrics.Metrics, error)
}

type MetricRepo struct {
	db map[string]metrics.Metrics
}

func NewMetricRepo() *MetricRepo {
	return &MetricRepo{
		db: make(map[string]metrics.Metrics),
	}
}

func (m *MetricRepo) SaveMetric(metric metrics.Metrics) error {
	if v, ok := m.db[metric.ID]; ok {
		newValue, err := metrics.NewValue(v, metric)
		if err != nil {
			return err
		}
		m.db[metric.ID] = newValue
	} else {
		m.db[metric.ID] = metric
	}
	return nil
}

func (m *MetricRepo) ListMetrics() []string {
	var list []string
	for k, v := range m.db {
		list = append(list, k+":"+v.ValueString())
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i] < list[j]
	})
	return list
}

func (m *MetricRepo) GetValue(metricName string) (metrics.Metrics, error) {
	v, ok := m.db[metricName]
	if !ok {
		return metrics.Metrics{}, errors.New("metric not stored")
	}
	return v, nil
}
