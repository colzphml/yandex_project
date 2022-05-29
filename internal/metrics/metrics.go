package metrics

import (
	"errors"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/colzphml/yandex_project/internal/utils"
)

type Gauge float64
type Counter int64

type MetricValue interface {
	String() string
	Type() string
}

func (g Gauge) String() string {
	return strconv.FormatFloat(float64(g), 'g', -1, 64)
}

func (g Gauge) Type() string {
	return "gauge"
}

func (c Counter) String() string {
	return strconv.FormatInt(int64(c), 10)
}

func (c Counter) Type() string {
	return "counter"
}

var (
	ErrUndefinedType = errors.New("type of metric undefined")
	ErrParseMetric   = errors.New("can't parse metric")
)

func GetRuntimeMetric(m *runtime.MemStats, fieldName string, fieldType string) (MetricValue, error) {
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	f := r.FieldByName(fieldName)
	if f == (reflect.Value{}) {
		return nil, errors.New("runtime not have this variable:" + fieldName + ", check config file")
	}
	switch t := r.FieldByName(fieldName).Type().Name(); {
	case strings.Contains(t, "int") && fieldType == "gauge":
		return Gauge(f.Uint()), nil
	case strings.Contains(t, "int") && fieldType == "counter":
		return Counter(f.Uint()), nil
	case strings.Contains(t, "float") && fieldType == "gauge":
		return Gauge(f.Float()), nil
	case strings.Contains(t, "float") && fieldType == "counter":
		return Counter(f.Float()), nil
	default:
		return nil, errors.New("not know type of variable: " + fieldType + ", check config file")
	}
}

func CollectMetrics(cfg *utils.AgentConfig, runtime *runtime.MemStats, inc Counter) map[string]MetricValue {
	metricsDescr := cfg.Metrics
	metricsStore := make(map[string]MetricValue)
	for k, v := range metricsDescr {
		value, err := GetRuntimeMetric(runtime, k, v)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		metricsStore[k] = value
	}
	metricsStore["PollCount"] = inc
	metricsStore["RandomValue"] = Gauge(rand.Float64())
	return metricsStore
}

func SendMetrics(cfg *utils.AgentConfig, input map[string]MetricValue, client *http.Client) {
	var urlPrefix, urlPart string
	urlPrefix = "http://" + cfg.ServerAdress + ":" + strconv.Itoa(cfg.ServerPort)
	for k, v := range input {
		urlPart = "/update/" + v.Type() + "/" + k + "/" + v.String()
		err := utils.HTTPSend(client, urlPrefix+urlPart)
		if err != nil {
			log.Println(err.Error())
			continue
		}
	}
}

func ConvertToMetric(metricType, metricValue string) (MetricValue, error) {
	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return Gauge(-1), ErrParseMetric
		}
		return Gauge(value), nil
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return Counter(-1), ErrParseMetric
		}
		return Counter(value), nil
	default:
		return Counter(-1), ErrUndefinedType
	}
}

func NewValue(oldValue MetricValue, newValue MetricValue) (MetricValue, error) {
	newValueType := newValue.Type()
	if oldValue.Type() != newValueType {
		return nil, errors.New("metric have another type")
	}
	switch newValueType {
	case "counter":
		newValue := oldValue.(Counter) + newValue.(Counter)
		return newValue, nil
	case "gauge":
		return newValue.(Gauge), nil
	default:
		return nil, ErrUndefinedType
	}
}
