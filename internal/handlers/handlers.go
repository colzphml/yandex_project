package handlers

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/go-chi/chi/v5"
)

func SaveHandler(repo *storage.MetricRepo) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "metric_type")
		metricName := chi.URLParam(r, "metric_name")
		metricValue := chi.URLParam(r, "metric_value")
		if metricName == "" || metricValue == "" {
			http.Error(rw, "can't parse metric: "+r.URL.Path, http.StatusNotFound)
			return
		}
		switch metricType {
		case "gauge":
			value, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(rw, "can't parse metric: "+r.URL.Path, http.StatusBadRequest)
				return
			}
			err = repo.SaveMetric(metricName, metrics.MetricValue{Type: metricType, Value: metrics.Gauge(value)})
			if err != nil {
				http.Error(rw, "can't save metric: "+r.URL.Path, http.StatusBadRequest)
				return
			}
		case "counter":
			value, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(rw, "can't parse metric: "+r.URL.Path, http.StatusBadRequest)
				return
			}
			err = repo.SaveMetric(metricName, metrics.MetricValue{Type: metricType, Value: metrics.Counter(value)})
			if err != nil {
				http.Error(rw, "can't save metric: "+r.URL.Path, http.StatusBadRequest)
				return
			}
		default:
			http.Error(rw, "undefined metric type: "+metricType, http.StatusNotImplemented)
			return
		}
		log.Println(repo)
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("Metric saved"))
	}
}

func ListMetrics(repo *storage.MetricRepo) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		metricList := repo.ListMetrics()
		_, err := io.WriteString(rw, strings.Join(metricList, ","))
		if err != nil {
			panic(err)
		}
	}
}

func GetValue(repo *storage.MetricRepo) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mName := chi.URLParam(r, "metric_name")
		mType := chi.URLParam(r, "metric_type")
		value, metricType, err := repo.GetValue(mName)
		if err != nil {
			http.Error(rw, "undefined metric: "+mName, http.StatusNotFound)
			return
		}
		if metricType != mType {
			http.Error(rw, "this metric have another type: "+mName, http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(value))
	}
}
