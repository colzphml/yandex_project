package handlers

import (
	"io"
	"net/http"
	"strings"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/go-chi/chi/v5"
)

func SaveHandler(repo storage.Repositories) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		metricType := chi.URLParam(r, "metric_type")
		metricName := chi.URLParam(r, "metric_name")
		metricValue := chi.URLParam(r, "metric_value")
		if metricName == "" || metricValue == "" {
			http.Error(rw, "can't parse metric: "+r.URL.Path, http.StatusNotFound)
			return
		}
		mValue, err := metrics.ConvertToMetric(metricType, metricValue)
		switch err {
		case metrics.ErrParseMetric:
			http.Error(rw, err.Error()+" "+r.URL.Path, http.StatusBadRequest)
			return
		case metrics.ErrUndefinedType:
			http.Error(rw, err.Error()+" "+r.URL.Path, http.StatusNotImplemented)
			return
		}
		err = repo.SaveMetric(metricName, mValue)
		if err != nil {
			http.Error(rw, "can't save metric: "+r.URL.Path, http.StatusBadRequest)
			return
		}
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte("Metric saved"))
	}
}

func ListMetricsHandler(repo storage.Repositories) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		metricList := repo.ListMetrics()
		_, err := io.WriteString(rw, strings.Join(metricList, "\n"))
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func GetValueHandler(repo storage.Repositories) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mName := chi.URLParam(r, "metric_name")
		mType := chi.URLParam(r, "metric_type")
		metricValue, err := repo.GetValue(mName)
		if err != nil {
			http.Error(rw, err.Error()+" "+mName, http.StatusNotFound)
			return
		}
		if metricValue.Type() != mType {
			http.Error(rw, "this metric have another type: "+mName, http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(metricValue.String()))
	}
}
