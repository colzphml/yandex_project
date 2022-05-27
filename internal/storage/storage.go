package storage

import (
	"errors"
	"sort"

	"github.com/colzphml/yandex_project/internal/metrics"
)

type Repositories interface {
	SaveMetric(metricName string, MetricValue metrics.MetricValue) error
	ListMetrics() []string
	GetValue(metricName string) (string, error)
}

type MetricRepo struct {
	db map[string]interface{}
}

func NewMetricRepo() *MetricRepo {
	return &MetricRepo{
		db: make(map[string]interface{}),
	}
}

func (m *MetricRepo) SaveMetric(metricName string, MetricValue metrics.MetricValue) error {
	switch MetricValue.Type {
	case "counter":
		if v, ok := m.db[metricName]; ok {
			m.db[metricName] = v.(metrics.Counter) + MetricValue.Value.(metrics.Counter)
			return nil
		} else {
			m.db[metricName] = MetricValue.Value.(metrics.Counter)
			return nil
		}
	case "gauge":
		m.db[metricName] = MetricValue.Value.(metrics.Gauge)
		return nil
	default:
		return errors.New("Metric " + metricName + "value not stored")
	}
}

func (m *MetricRepo) ListMetrics() []string {
	var list []string
	for k := range m.db {
		list = append(list, k)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i] < list[j]
	})
	return list
}

func (m *MetricRepo) GetValue(metricName string) (string, string, error) {
	v, ok := m.db[metricName]
	if !ok {
		return "", "", errors.New("metric not stored")
	}
	switch m.db[metricName].(type) {
	case metrics.Gauge:
		return v.(metrics.Gauge).String(), "gauge", nil
	case metrics.Counter:
		return v.(metrics.Counter).String(), "counter", nil
	default:
		return "", "", errors.New("metric stored, but type undefined")
	}
}
