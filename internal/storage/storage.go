package storage

import (
	"errors"

	"github.com/colzphml/yandex_project/internal/metrics"
)

type Repositories interface {
	SaveMetric(metricName string, MetricValue metrics.MetricValue) error
}

type MetricRepo struct {
	db map[string]interface{}
}

func NewMetricRepo() MetricRepo {
	return MetricRepo{
		db: make(map[string]interface{}),
	}
}

func (m MetricRepo) SaveMetric(metricName string, MetricValue metrics.MetricValue) error {
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
