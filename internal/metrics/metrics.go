package metrics

import (
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

type RuntimeMetrics struct {
	Alloc         Gauge
	BuckHashSys   Gauge
	Frees         Gauge
	GCCPUFraction Gauge
	GCSys         Gauge
	HeapAlloc     Gauge
	HeapIdle      Gauge
	HeapInuse     Gauge
	HeapObjects   Gauge
	HeapReleased  Gauge
	HeapSys       Gauge
	LastGC        Gauge
	Lookups       Gauge
	MCacheInuse   Gauge
	MCacheSys     Gauge
	MSpanInuse    Gauge
	MSpanSys      Gauge
	Mallocs       Gauge
	NextGC        Gauge
	NumForcedGC   Gauge
	NumGC         Gauge
	OtherSys      Gauge
	PauseTotalNs  Gauge
	StackInuse    Gauge
	StackSys      Gauge
	Sys           Gauge
	TotalAlloc    Gauge
}

type AdditionalMetrics struct {
	PollCount   Counter
	RandomValue Gauge
}

func getRuntimeMetric(m runtime.MemStats, field string) Gauge {
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	f := r.FieldByName(field)
	switch t := r.FieldByName(field).Type().Name(); {
	case strings.Contains(t, "int"):
		return Gauge(f.Uint())
	case strings.Contains(t, "float"):
		return Gauge(f.Float())
	default:
		log.Printf("unknown type of metric: %v", r.FieldByName(field).Type().Name())
	}
	return Gauge(f.Float())
}

func SetMetrics(runtime *RuntimeMetrics, addit *AdditionalMetrics, runtimeState runtime.MemStats, inc Counter) {
	r := reflect.ValueOf(runtime)
	if r.Kind() == reflect.Ptr {
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
	addit.RandomValue = Gauge(rand.Float64())
}

func SendMetrics(cfg *utils.AgentConfig, input interface{}, client http.Client) {
	var r reflect.Value
	switch input := input.(type) {
	case RuntimeMetrics:
		metrics := input
		r = reflect.ValueOf(&metrics)
	case AdditionalMetrics:
		metrics := input
		r = reflect.ValueOf(&metrics)
	}
	urlPrefix := fmt.Sprintf("http://%v:%v/update", cfg.ServerAdress, cfg.ServerPort)
	urlPart := ""
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	for j := 0; j < r.NumField(); j++ {
		switch r.Field(j).Type().Name() {
		case "Gauge":
			urlPart = fmt.Sprintf("/%v/%v/%v", r.Field(j).Type().Name(), r.Type().Field(j).Name, r.Field(j).Float())
		case "Counter":
			urlPart = fmt.Sprintf("/%v/%v/%v", r.Field(j).Type().Name(), r.Type().Field(j).Name, r.Field(j).Int())
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
		response.Body.Close()
	}
}
