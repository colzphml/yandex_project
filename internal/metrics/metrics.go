package metrics

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/colzphml/yandex_project/internal/utils"
)

type Gauge float64
type Counter int64

type MetricValue struct {
	Value interface{}
	Type  string
}

func GetRuntimeMetric(m *runtime.MemStats, fieldName string, fieldType string) (interface{}, error) {
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
		metricsStore[k] = MetricValue{Value: value, Type: v}
	}
	metricsStore["PollCount"] = MetricValue{Value: inc, Type: "counter"}
	metricsStore["RandomValue"] = MetricValue{Value: rand.Float64(), Type: "gauge"}
	return metricsStore
}

func SendMetrics(cfg *utils.AgentConfig, input map[string]MetricValue, client *http.Client) {
	var urlPrefix, urlPart string
	urlPrefix = fmt.Sprintf("http://%v:%v/update", cfg.ServerAdress, cfg.ServerPort)
	for k, v := range input {
		urlPart = fmt.Sprintf("/%v/%v/%v/", v.Type, k, v.Value)
		err := utils.HTTPSend(client, urlPrefix+urlPart)
		if err != nil {
			log.Println(err.Error())
			continue
		}
	}
}
