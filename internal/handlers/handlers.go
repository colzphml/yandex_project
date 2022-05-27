package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/storage"
)

func SaveHandler(repo *storage.MetricRepo) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method != http.MethodPost:
			http.Error(rw, "request is not POST", http.StatusBadRequest)
			return
			//...maybe more cases
		default:
			input := strings.Split(r.URL.Path, "/")
			if len(input) < 5 {
				http.Error(rw, "can't parse metric: "+r.URL.Path, http.StatusNotFound)
				return
			}
			switch input[2] {
			case "gauge":
				value, err := strconv.ParseFloat(input[4], 64)
				if err != nil {
					http.Error(rw, "can't parse metric: "+r.URL.Path, http.StatusBadRequest)
					return
				}
				err = repo.SaveMetric(input[3], metrics.MetricValue{Type: input[2], Value: metrics.Gauge(value)})
				if err != nil {
					http.Error(rw, "can't save metric: "+r.URL.Path, http.StatusBadRequest)
					return
				}
			case "counter":
				value, err := strconv.ParseInt(input[4], 10, 64)
				if err != nil {
					http.Error(rw, "can't parse metric: "+r.URL.Path, http.StatusBadRequest)
					return
				}
				err = repo.SaveMetric(input[3], metrics.MetricValue{Type: input[2], Value: metrics.Counter(value)})
				if err != nil {
					http.Error(rw, "can't save metric: "+r.URL.Path, http.StatusBadRequest)
					return
				}
			default:
				http.Error(rw, "undefined metric type: "+input[2], http.StatusNotImplemented)
				return
			}
			rw.Header().Set("Content-Type", "test/plain")
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte("Metric saved"))
		}
	}
}
