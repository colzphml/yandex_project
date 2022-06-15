package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/colzphml/yandex_project/internal/utils_server"
	"github.com/go-chi/chi/v5"
)

func SaveHandler(repo storage.Repositorier) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		metricName := chi.URLParam(r, "metric_name")
		metricType := chi.URLParam(r, "metric_type")
		metricValue := chi.URLParam(r, "metric_value")
		if metricName == "" || metricValue == "" {
			http.Error(rw, "can't parse metric: "+r.URL.Path, http.StatusNotFound)
			return
		}
		mValue, err := metrics.ConvertToMetric(metricName, metricType, metricValue)
		switch err {
		case metrics.ErrParseMetric:
			http.Error(rw, err.Error()+" "+r.URL.Path, http.StatusBadRequest)
			return
		case metrics.ErrUndefinedType:
			http.Error(rw, err.Error()+" "+r.URL.Path, http.StatusNotImplemented)
			return
		}
		err = repo.SaveMetric(mValue)
		if err != nil {
			http.Error(rw, "can't save metric: "+r.URL.Path, http.StatusBadRequest)
			return
		}
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("Metric saved"))
	}
}

func SaveJSONHandler(repo storage.Repositorier, cfg *utils_server.ServerConfig) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var m metrics.Metrics
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(rw, "can't decode metric: "+r.URL.Path, http.StatusBadRequest)
			return
		}
		err := repo.SaveMetric(m)
		if err != nil {
			http.Error(rw, "can't save metric: "+m.ID, http.StatusBadRequest)
			return
		}
		if cfg.StoreInterval == 0*time.Second {
			err = repo.StoreMetric(cfg)
			if err != nil {
				http.Error(rw, "can't store metric: "+m.ID, http.StatusInternalServerError)
				return
			}
		}
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("Metric saved"))
	}
}

func ListMetricsHandler(repo storage.Repositorier, cfg *utils_server.ServerConfig) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		metricList := repo.ListMetrics()
		_, err := io.WriteString(rw, strings.Join(metricList, "\n"))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func GetValueHandler(repo storage.Repositorier) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mName := chi.URLParam(r, "metric_name")
		mType := chi.URLParam(r, "metric_type")
		metricValue, err := repo.GetValue(mName)
		if err != nil {
			http.Error(rw, err.Error()+" "+mName, http.StatusNotFound)
			return
		}
		if metricValue.MType != mType {
			http.Error(rw, "this metric have another type: "+mName, http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(metricValue.ValueString()))
	}
}

func GetJSONValueHandler(repo storage.Repositorier) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var m metrics.Metrics
		if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
			http.Error(rw, "can't decode metric: "+r.URL.Path, http.StatusBadRequest)
			return
		}
		metricValue, err := repo.GetValue(m.ID)
		if err != nil {
			http.Error(rw, err.Error()+" "+m.ID, http.StatusNotFound)
			return
		}
		if metricValue.MType != m.MType {
			http.Error(rw, "this metric have another type: "+m.ID, http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusOK)
		js, err := json.Marshal(metricValue)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
		rw.Write(js)
	}
}
