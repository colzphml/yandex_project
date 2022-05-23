package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"syscall"
	"time"
)

type gauge float64
type counter int64

const pollInterval = 2
const reportInterval = 10
const serverAdress = "127.0.0.1"
const serverPort = 8080

type runtimeMetrics struct {
	Alloc       gauge
	BuckHashSys gauge
}

type additionalMetrics struct {
	PollCount   counter
	RandomValue gauge
}

var runtimeState runtime.MemStats

func getValueByField(m *runtime.MemStats, field string) gauge {
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	f := r.FieldByName(field)
	return gauge(f.Uint())
}

func setMetrics(runtime *runtimeMetrics, addit *additionalMetrics, inc counter) {
	r := reflect.ValueOf(runtime)
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	for j := 0; j < r.NumField(); j++ {
		switch r.Field(j).Type().Name() {
		case "gauge":
			r.Field(j).SetFloat(float64(getValueByField(&runtimeState, r.Type().Field(j).Name)))
		case "counter":
			r.Field(j).SetInt(int64(getValueByField(&runtimeState, r.Type().Field(j).Name)))
		default:
			fmt.Println("undefined type")
		}
	}
	addit.PollCount = inc
	addit.RandomValue = gauge(rand.Float64())
}

func sendMetrics(input interface{}, client http.Client) {
	var r reflect.Value
	switch input.(type) {
	case runtimeMetrics:
		metrics := input.(runtimeMetrics)
		r = reflect.ValueOf(&metrics)
	case additionalMetrics:
		metrics := input.(additionalMetrics)
		r = reflect.ValueOf(&metrics)
	}
	urlPrefix := fmt.Sprintf("http://%v:%v/update", serverAdress, serverPort)
	urlPart := ""
	if r.Kind() == reflect.Pointer {
		r = r.Elem()
	}
	for j := 0; j < r.NumField(); j++ {
		switch r.Field(j).Type().Name() {
		case "gauge":
			urlPart = fmt.Sprintf("/%v/%v/%v", r.Field(j).Type().Name(), r.Type().Field(j).Name, gauge(r.Field(j).Float()))
		case "counter":
			urlPart = fmt.Sprintf("/%v/%v/%v", r.Field(j).Type().Name(), r.Type().Field(j).Name, counter(r.Field(j).Int()))
		default:
			log.Println("undefined type")
			continue
		}
		request, err := http.NewRequest(http.MethodPost, urlPrefix+urlPart, nil)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		request.Header.Set("Content-Type", "text/plain")
		/*_, err = client.Do(request)
		if err != nil {
			log.Println(err.Error())
		}*/
		fmt.Println(request)
	}
}

func main() {
	//for metrics value
	var currentRntState runtimeMetrics
	var currentAddState additionalMetrics
	//for close programm by signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	//for additional metric PollCount
	var pollCouter counter = 0
	//for additional metric RandomValue
	rand.Seed(time.Now().UnixNano())
	//can we get collision every 5th tickerGet and every tickerSend??? Maybe send in other goroutine??
	tickerGet := time.NewTicker(pollInterval * time.Second)
	tickerSend := time.NewTicker(reportInterval * time.Second)
	//client for send
	client := http.Client{}
	//maybe there is a better way
out:
	for {
		select {
		case <-tickerGet.C:
			runtime.ReadMemStats(&runtimeState)
			setMetrics(&currentRntState, &currentAddState, pollCouter)
			pollCouter++
		case <-tickerSend.C:
			sendMetrics(currentRntState, client)
			sendMetrics(currentAddState, client)
		case <-sigChan:
			fmt.Println("close program")
			break out
		}
	}
}
