// Package handlers описывает логику работы endpoints.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/colzphml/yandex_project/internal/app/server/serverutils"
	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/metrics/metricsserver"
	"github.com/colzphml/yandex_project/internal/scenarios"
	"github.com/colzphml/yandex_project/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

var log = zerolog.New(serverutils.LogConfig()).With().Timestamp().Str("component", "handlers").Logger()

type Handlers struct {
	repo storage.Repositorier
	cfg  *serverutils.ServerConfig
}

func New(ctx context.Context, repo storage.Repositorier, cfg *serverutils.ServerConfig) *Handlers {
	result := &Handlers{
		repo: repo,
		cfg:  cfg,
	}
	return result
}

func errMapping(err error) int {
	switch {
	case errors.Is(err, scenarios.ErrStatusBadRequest):
		return http.StatusBadRequest
	case errors.Is(err, scenarios.ErrStatusNotFound):
		return http.StatusNotFound
	case errors.Is(err, scenarios.ErrStatusNotImplemented):
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

// SaveHandler - хэндлер, сохраняющий метрику из URL.
//
// POST [/update/{metric_type}/{metric_name}/{metric_value}].
func (h Handlers) SaveHandler(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metricName := chi.URLParam(r, "metric_name")
	metricType := chi.URLParam(r, "metric_type")
	metricValue := chi.URLParam(r, "metric_value")
	if metricName == "" || metricValue == "" {
		http.Error(rw, "can't parse metric: "+r.URL.Path, http.StatusNotFound)
		return
	}
	mValue, err := metricsserver.ConvertToMetric(metricName, metricType, metricValue)
	switch {
	case errors.Is(err, metrics.ErrParseMetric):
		http.Error(rw, err.Error()+" "+r.URL.Path, http.StatusBadRequest)
		return
	case errors.Is(err, metrics.ErrUndefinedType):
		http.Error(rw, err.Error()+" "+r.URL.Path, http.StatusNotImplemented)
		return
	}
	err = scenarios.SaveMetric(ctx, h.repo, h.cfg, mValue, false)
	if err != nil {
		http.Error(rw, err.Error()+" "+r.URL.Path, errMapping(err))
		return
	}
	rw.Header().Set("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("Metric saved"))
}

// SaveJSONHandler - хэндлер, сохраняющий метрику из body в формате JSON. Проверяет подпись данных.
//
// POST [/update/].
func (h Handlers) SaveJSONHandler(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var m metrics.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(rw, "can't decode metric: "+r.URL.Path, http.StatusBadRequest)
		return
	}
	err := scenarios.SaveMetric(ctx, h.repo, h.cfg, m, true)
	if err != nil {
		http.Error(rw, err.Error()+" "+r.URL.Path, errMapping(err))
		return
	}
	rw.Header().Set("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte("Metric saved"))
}

// SaveJSONArrayHandler - хэндлер, сохраняющий массив метрик из body в формате JSON. Проверяет подпись данных.
//
// POST [/updates/].
func (h Handlers) SaveJSONArrayHandler(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var m []metrics.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(rw, "can't decode metric: "+r.URL.Path, http.StatusBadRequest)
		return
	}
	count, err := scenarios.SaveArrayMetric(ctx, h.repo, h.cfg, m)
	if err != nil {
		http.Error(rw, err.Error(), errMapping(err))
		return
	}
	rw.Header().Set("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	fmt.Fprintf(rw, "Metric saved, count: %d", count)
	//rw.Write([]byte("Metric saved, count: " + strconv.Itoa(count)))
}

// ListMetricsHandler - возвращает список сохраненных метрик с их значением.
//
// GET [/].
func (h Handlers) ListMetricsHandler(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metricList := h.repo.ListMetrics(ctx)
	var result []string
	for _, v := range metricList {
		result = append(result, v.ID+":"+v.ValueString())
	}
	rw.Header().Set("Content-Type", "text/html")
	rw.WriteHeader(http.StatusOK)
	_, err := io.WriteString(rw, strings.Join(result, "<br>"))
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetValueHandler - возвращает значение метрики для запрошенного имени.
//
// GET [/value/{metric_type}/{metric_name}].
func (h Handlers) GetValueHandler(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mName := chi.URLParam(r, "metric_name")
	mType := chi.URLParam(r, "metric_type")
	metricValue, err := scenarios.GetMetric(ctx, h.repo, h.cfg, mName, mType, false)
	if err != nil {
		http.Error(rw, err.Error(), errMapping(err))
		return
	}
	rw.Header().Set("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	fmt.Fprint(rw, metricValue.ValueString())
	//rw.Write([]byte(metricValue.ValueString()))
}

// GetJSONValueHandler - возвращает метрику для запрошенного имени в формате JSON.
//
// GET [/value/].
func (h Handlers) GetJSONValueHandler(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var m metrics.Metrics
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(rw, "can't decode metric: "+r.URL.Path, http.StatusBadRequest)
		return
	}
	metricValue, err := scenarios.GetMetric(ctx, h.repo, h.cfg, m.ID, m.MType, true)
	if err != nil {
		http.Error(rw, err.Error(), errMapping(err))
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

// PingHandler - проверяет доступность хранилища.
//
// GET [/ping].
func (h Handlers) PingHandler(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	err := h.repo.Ping(ctx)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	fmt.Fprint(rw, "ok")
	//rw.Write([]byte("ok"))
}
