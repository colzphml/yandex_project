package metricsagent

import (
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/colzphml/yandex_project/internal/agentutils"
	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/rs/zerolog"
)

var log = zerolog.New(agentutils.LogConfig()).With().Timestamp().Str("component", "metricsagent").Logger()

func GetRuntimeMetric(m *runtime.MemStats, fieldName string, fieldType string) (metrics.Metrics, error) {
	var result metrics.Metrics
	result.ID = fieldName
	result.MType = fieldType
	r := reflect.ValueOf(m)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	f := r.FieldByName(fieldName)
	if f == (reflect.Value{}) {
		return metrics.Metrics{}, errors.New("runtime not have this variable:" + fieldName + ", check config file")
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
		return metrics.Metrics{}, errors.New("not know type of variable: " + fieldType + ", check config file")
	}
}

func CollectMetrics(metricsDescr map[string]string, runtime *runtime.MemStats, inc int64) map[string]metrics.Metrics {
	metricsStore := make(map[string]metrics.Metrics)
	for k, v := range metricsDescr {
		value, err := GetRuntimeMetric(runtime, k, v)
		if err != nil {
			log.Error().Err(err).Msg("failed collect metric")
			continue
		}
		metricsStore[k] = value
	}
	incMetrics := metrics.Metrics{ID: "PollCount", MType: "counter", Delta: &inc}
	metricsStore[incMetrics.ID] = incMetrics
	randomValue := rand.Float64()
	randMetrics := metrics.Metrics{ID: "RandomValue", MType: "gauge", Value: &randomValue}
	metricsStore[randMetrics.ID] = randMetrics
	return metricsStore
}

func SendMetrics(srv string, input map[string]metrics.Metrics, client *http.Client) {
	var urlPrefix, urlPart string
	urlPrefix = "http://" + srv
	for k, v := range input {
		urlPart = "/update/" + v.MType + "/" + k + "/" + v.ValueString()
		err := agentutils.HTTPSend(client, urlPrefix+urlPart)
		if err != nil {
			log.Error().Err(err).Msg("failed send metrics by url")
			continue
		}
	}
}

func SendJSONMetrics(srv string, key string, input map[string]metrics.Metrics, client *http.Client) {
	urlPrefix := "http://" + srv + "/update/"
	for _, v := range input {
		v.FillHash(key)
		postBody, err := json.Marshal(v)
		if err != nil {
			log.Error().Err(err).Msg("failed marshall json")
			continue
		}
		err = agentutils.HTTPSendJSON(client, urlPrefix, postBody)
		if err != nil {
			log.Error().Err(err).Msg("failed send with body")
			continue
		}
	}
}

func SendListJSONMetrics(srv string, key string, input map[string]metrics.Metrics, client *http.Client) {
	urlPrefix := "http://" + srv + "/updates/"
	var list []metrics.Metrics
	for _, v := range input {
		v.FillHash(key)
		list = append(list, v)
	}
	postBody, err := json.Marshal(list)
	if err != nil {
		log.Error().Err(err).Msg("failed marshall json")
	}
	err = agentutils.HTTPSendJSON(client, urlPrefix, postBody)
	if err != nil {
		log.Error().Err(err).Msg("failed send with body (list)")
	}
}
