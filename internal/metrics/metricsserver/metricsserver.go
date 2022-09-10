// Модуль metricsserver содержит специфические для сервера методы по работе с метриками.
package metricsserver

import (
	"strconv"

	"github.com/colzphml/yandex_project/internal/metrics"
)

// ConvertToMetric - превращает строковые значения имени, типа и значения в метрику.
func ConvertToMetric(metricName, metricType, metricValue string) (metrics.Metrics, error) {
	var result metrics.Metrics
	result.ID = metricName
	result.MType = metricType
	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return metrics.Metrics{}, metrics.ErrParseMetric
		}
		result.Value = &value
		return result, nil
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return metrics.Metrics{}, metrics.ErrParseMetric
		}
		result.Delta = &value
		return result, nil
	default:
		return metrics.Metrics{}, metrics.ErrUndefinedType
	}
}

// NewValue - логическая операция по обновлению метрики: gauge перезаписывается, counter суммируется с предыдущим значением.
func NewValue(oldValue metrics.Metrics, newValue metrics.Metrics) (metrics.Metrics, error) {
	var result metrics.Metrics
	result.ID = newValue.ID
	if oldValue.MType != newValue.MType {
		return metrics.Metrics{}, metrics.ErrWrongType
	}
	result.MType = newValue.MType
	switch newValue.MType {
	case "counter":
		newValue := *oldValue.Delta + *newValue.Delta
		result.Delta = &newValue
		return result, nil
	case "gauge":
		newValue := *newValue.Value
		result.Value = &newValue
		return result, nil
	default:
		return metrics.Metrics{}, metrics.ErrUndefinedType
	}
}
