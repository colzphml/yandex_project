package metrics

import (
	"encoding/json"
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

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

func (m Metrics) ValueString() string {
	switch m.MType {
	case "gauge":
		return strconv.FormatFloat(float64(*m.Value), 'g', -1, 64)
	case "counter":
		return strconv.FormatInt(int64(*m.Delta), 10)
	default:
		return ""
	}
}

func NewMetrics(name string, value MetricValue) (Metrics, error) {
	var result Metrics
	result.ID = name
	result.MType = value.Type()
	switch result.MType {
	case "gauge":
		var v float64
		v = float64(value.(Gauge))
		result.Value = &v
	case "counter":
		var v int64
		v = int64(value.(Counter))
		result.Delta = &v
	default:
		return Metrics{}, ErrUndefinedType
	}

	return result, nil
}

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

func GetRuntimeMetric(m *runtime.MemStats, fieldName string, fieldType string) (Metrics, error) {
	var result Metrics
	result.ID = fieldName
	result.MType = fieldType
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	f := r.FieldByName(fieldName)
	if f == (reflect.Value{}) {
		return Metrics{}, errors.New("runtime not have this variable:" + fieldName + ", check config file")
	}
	switch t := r.FieldByName(fieldName).Type().Name(); {
	case strings.Contains(t, "int") && fieldType == "gauge":
		var v float64
		v = float64(f.Uint())
		result.Value = &v
		return result, nil
	case strings.Contains(t, "int") && fieldType == "counter":
		var v int64
		v = int64(f.Uint())
		result.Delta = &v
		return result, nil
	case strings.Contains(t, "float") && fieldType == "gauge":
		var v float64
		v = float64(f.Float())
		result.Value = &v
		return result, nil
	case strings.Contains(t, "float") && fieldType == "counter":
		var v int64
		v = int64(f.Float())
		result.Delta = &v
		return result, nil
	default:
		return Metrics{}, errors.New("not know type of variable: " + fieldType + ", check config file")
	}
}

func CollectMetrics(cfg *utils.AgentConfig, runtime *runtime.MemStats, inc int64) map[string]Metrics {
	metricsDescr := cfg.Metrics
	metricsStore := make(map[string]Metrics)
	for k, v := range metricsDescr {
		value, err := GetRuntimeMetric(runtime, k, v)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		metricsStore[k] = value
	}
	//int64Value := inc
	incMetrics := Metrics{ID: "PollCount", MType: "counter", Delta: &inc}
	metricsStore[incMetrics.ID] = incMetrics
	randomValue := rand.Float64()
	randMetrics := Metrics{ID: "RandomValue", MType: "gauge", Value: &randomValue}
	metricsStore[randMetrics.ID] = randMetrics
	return metricsStore
}

func SendMetrics(cfg *utils.AgentConfig, input map[string]Metrics, client *http.Client) {
	var urlPrefix, urlPart string
	urlPrefix = "http://" + cfg.ServerAdress + ":" + strconv.Itoa(cfg.ServerPort)
	for k, v := range input {
		urlPart = "/update/" + v.MType + "/" + k + "/" + v.ValueString()
		err := utils.HTTPSend(client, urlPrefix+urlPart)
		if err != nil {
			log.Println(err.Error())
			continue
		}
	}
}

func SendJSONMetrics(cfg *utils.AgentConfig, input map[string]Metrics, client *http.Client) {
	//var urlPrefix, urlPart string
	//urlPrefix = "http://" + cfg.ServerAdress + ":" + strconv.Itoa(cfg.ServerPort)
	for _, v := range input {
		js, err := json.MarshalIndent(v, "", "	")
		if err != nil {
			log.Println(err.Error())
			continue
		}
		log.Println(string(js))
	}
}

func ConvertToMetric(metricName, metricType, metricValue string) (Metrics, error) {
	var result Metrics
	result.ID = metricName
	result.MType = metricType
	switch metricType {
	case "gauge":
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return Metrics{}, ErrParseMetric
		}
		result.Value = &value
		return result, nil
	case "counter":
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return Metrics{}, ErrParseMetric
		}
		result.Delta = &value
		return result, nil
	default:
		return Metrics{}, ErrUndefinedType
	}
}

func NewValue(oldValue Metrics, newValue Metrics) (Metrics, error) {
	var result Metrics
	result.ID = newValue.ID
	if oldValue.MType != newValue.MType {
		return Metrics{}, errors.New("metric have another type")
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
		return Metrics{}, ErrUndefinedType
	}
}
