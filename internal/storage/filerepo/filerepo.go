package filerepo

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"sort"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/serverutils"
)

var (
	ErrNoFileDeclared = errors.New("RESTORE == true, but STORE_FILE is empty")
)

type MetricRepo struct {
	DB map[string]metrics.Metrics
}

func NewMetricRepo(cfg *serverutils.ServerConfig) (*MetricRepo, error) {
	repo := &MetricRepo{
		DB: make(map[string]metrics.Metrics),
	}
	switch {
	case cfg.Restore && cfg.StoreFile != "":
		file, err := os.OpenFile(cfg.StoreFile, os.O_RDONLY|os.O_CREATE, 0777)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		err = json.NewDecoder(file).Decode(repo)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return repo, nil
			}
			return nil, err
		}
		return repo, nil
	case cfg.Restore && cfg.StoreFile == "":
		return nil, ErrNoFileDeclared
	default:
		return repo, nil
	}
}

func (m *MetricRepo) DumpMetrics(cfg *serverutils.ServerConfig) error {
	if cfg.StoreFile != "" {
		file, err := os.OpenFile(cfg.StoreFile, os.O_RDWR|os.O_CREATE, 0777)
		if err != nil {
			return err
		}
		defer file.Close()
		return json.NewEncoder(file).Encode(m)
	}
	return nil
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

func (m *MetricRepo) SaveListMetric(metricarray []metrics.Metrics) (int, error) {
	counter := 0
	for _, metric := range metricarray {
		if v, ok := m.DB[metric.ID]; ok {
			newValue, err := metrics.NewValue(v, metric)
			if err != nil {
				log.Println(err)
				continue
			}
			m.DB[metric.ID] = newValue
		} else {
			m.DB[metric.ID] = metric
		}
		counter++
	}
	return counter, nil
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

func (m *MetricRepo) Close() {
	//может быть в будущем будет пересмотрена работа с файлами
}

func (m *MetricRepo) Ping() error {
	return nil
}
