package metrics

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/colzphml/yandex_project/internal/agentutils"
)

var (
	ErrUndefinedType = errors.New("type of metric undefined")
	ErrParseMetric   = errors.New("can't parse metric")
	ErrWrongType     = errors.New("metric have another type")
)

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // значение хеш-функции
}

func (m *Metrics) ValueString() string {
	switch m.MType {
	case "gauge":
		return strconv.FormatFloat(float64(*m.Value), 'g', -1, 64)
	case "counter":
		return strconv.FormatInt(int64(*m.Delta), 10)
	default:
		return ""
	}
}

func (m *Metrics) CalculateHash(key string) ([]byte, error) {
	var src string
	switch m.MType {
	case "gauge":
		src = fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)
	case "counter":
		src = fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)
	default:
		return nil, ErrUndefinedType
	}
	hash, err := signData(src, key)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return hash, nil
}

func (m *Metrics) FillHash(key string) error {
	if key != "" {
		hash, err := m.CalculateHash(key)
		if err != nil {
			return err
		}
		m.Hash = hex.EncodeToString(hash)
	}
	return nil
}

func (m *Metrics) CompareHash(key string) (bool, error) {
	if key != "" {
		hash, err := m.CalculateHash(key)
		if err != nil {
			return false, err
		}
		data, err := hex.DecodeString(m.Hash)
		if err != nil {
			return false, err
		}
		return bytes.Equal(hash, data), nil
	}
	return true, nil
}

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

func CollectMetrics(cfg *agentutils.AgentConfig, runtime *runtime.MemStats, inc int64) map[string]Metrics {
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
	incMetrics := Metrics{ID: "PollCount", MType: "counter", Delta: &inc}
	metricsStore[incMetrics.ID] = incMetrics
	randomValue := rand.Float64()
	randMetrics := Metrics{ID: "RandomValue", MType: "gauge", Value: &randomValue}
	metricsStore[randMetrics.ID] = randMetrics
	return metricsStore
}

func SendMetrics(cfg *agentutils.AgentConfig, input map[string]Metrics, client *http.Client) {
	var urlPrefix, urlPart string
	urlPrefix = "http://" + cfg.ServerAddress
	for k, v := range input {
		urlPart = "/update/" + v.MType + "/" + k + "/" + v.ValueString()
		err := agentutils.HTTPSend(client, urlPrefix+urlPart)
		if err != nil {
			log.Println(err.Error())
			continue
		}
	}
}

func SendJSONMetrics(cfg *agentutils.AgentConfig, input map[string]Metrics, client *http.Client) {
	urlPrefix := "http://" + cfg.ServerAddress + "/update/"
	for _, v := range input {
		v.FillHash(cfg.Key)
		postBody, err := json.Marshal(v)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		err = agentutils.HTTPSendJSON(client, urlPrefix, postBody)
		if err != nil {
			log.Println(err.Error())
			continue
		}
	}
}

func SendListJSONMetrics(cfg *agentutils.AgentConfig, input map[string]Metrics, client *http.Client) {
	urlPrefix := "http://" + cfg.ServerAddress + "/updates/"
	var list []Metrics
	for _, v := range input {
		v.FillHash(cfg.Key)
		list = append(list, v)
	}
	postBody, err := json.Marshal(list)
	if err != nil {
		log.Println(err.Error())
	}
	err = agentutils.HTTPSendJSON(client, urlPrefix, postBody)
	if err != nil {
		log.Println(err.Error())
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
		return Metrics{}, ErrWrongType
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

func signData(src, key string) ([]byte, error) {
	h := hmac.New(sha256.New, []byte(key))
	_, err := h.Write([]byte(src))
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
