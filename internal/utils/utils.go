package utils

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/colzphml/yandex_project/internal/metrics"
)

func getRuntimeMetric(m runtime.MemStats, field string) metrics.Gauge {
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	f := r.FieldByName(field)
	switch t := r.FieldByName(field).Type().Name(); {
	case strings.Contains(t, "int"):
		return metrics.Gauge(f.Uint())
	case strings.Contains(t, "float"):
		return metrics.Gauge(f.Float())
	default:
		log.Printf("unknown type of metric: %v", r.FieldByName(field).Type().Name())
	}
	return metrics.Gauge(f.Float())
}

func SetMetrics(runtime *metrics.RuntimeMetrics, addit *metrics.AdditionalMetrics, runtimeState runtime.MemStats, inc metrics.Counter) {
	r := reflect.ValueOf(runtime)
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	for j := 0; j < r.NumField(); j++ {
		switch r.Field(j).Type().Name() {
		case "Gauge":
			r.Field(j).SetFloat(float64(getRuntimeMetric(runtimeState, r.Type().Field(j).Name)))
		case "Counter":
			r.Field(j).SetInt(int64(getRuntimeMetric(runtimeState, r.Type().Field(j).Name)))
		default:
			log.Println("undefined type for get")
		}
	}
	addit.PollCount = inc
	addit.RandomValue = metrics.Gauge(rand.Float64())
}

func SendMetrics(input interface{}, client http.Client) {
	var r reflect.Value
	switch input.(type) {
	case metrics.RuntimeMetrics:
		metrics := input.(metrics.RuntimeMetrics)
		r = reflect.ValueOf(&metrics)
	case metrics.AdditionalMetrics:
		metrics := input.(metrics.AdditionalMetrics)
		r = reflect.ValueOf(&metrics)
	}
	urlPrefix := fmt.Sprintf("http://%v:%v/update", metrics.ServerAdress, metrics.ServerPort)
	urlPart := ""
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	for j := 0; j < r.NumField(); j++ {
		switch r.Field(j).Type().Name() {
		case "Gauge":
			urlPart = fmt.Sprintf("/%v/%v/%v", r.Field(j).Type().Name(), r.Type().Field(j).Name, metrics.Gauge(r.Field(j).Float()))
		case "Counter":
			urlPart = fmt.Sprintf("/%v/%v/%v", r.Field(j).Type().Name(), r.Type().Field(j).Name, metrics.Counter(r.Field(j).Int()))
		default:
			log.Println("undefined type for send")
			continue
		}
		request, err := http.NewRequest(http.MethodPost, urlPrefix+urlPart, nil)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		request.Header.Set("Content-Type", "text/plain")
		response, err := client.Do(request)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		if response.StatusCode != 200 {
			log.Println("send failed")
			continue
		}
	}
}
