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
	db map[string]metrics.MetricValue
}

func NewMetricRepo() *MetricRepo {
	return &MetricRepo{
		db: make(map[string]metrics.MetricValue),
	}
}

func (m *MetricRepo) SaveMetric(metricName string, mValue metrics.MetricValue) error {
	if v, ok := m.db[metricName]; ok {
		newValue, err := metrics.NewValue(v, mValue)
		if err != nil {
			return err
		}
		m.db[metricName] = newValue
	} else {
		m.db[metricName] = mValue
	}
	return nil
}

func (m *MetricRepo) ListMetrics() []string {
	var list []string
	for k, v := range m.db {
		list = append(list, k+":"+v.String())
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
	mType := v.Type()
	mValue := v.String()
	return mValue, mType, nil
}
