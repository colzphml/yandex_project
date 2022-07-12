package dbrepo

import (
	"context"
	"errors"
	"sort"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
)

var log = zerolog.New(serverutils.LogConfig()).With().Timestamp().Str("component", "dbrepo").Logger()

var (
	SQLCreateTable        = "CREATE TABLE IF NOT EXISTS public.metrics (id varchar(50) NOT NULL,mtype varchar(50) NULL,delta int8 NULL,value float8 NULL,CONSTRAINT metrics_pkey PRIMARY KEY (id));"
	SQLTruncateTable      = "TRUNCATE TABLE public.metrics"
	SQLInsertGaugeValue   = "insert into metrics (id, mtype , value) values ($1,'gauge', $2) on conflict (id) do update set value = EXCLUDED.value"
	SQLInsertCounterValue = "insert into metrics (id, mtype , delta) values ($1,'counter', $2) on conflict (id) do update set delta = EXCLUDED.delta + metrics.delta"
	SQLSelectValueType    = "SELECT mtype FROM public.metrics where id = $1"
	SQLSelectValue        = "SELECT id, mtype, value, delta FROM public.metrics where id = $1"
	SQLSelectAllValues    = "SELECT id, mtype, value, delta FROM  public.metrics"
)

type MetricRepo struct {
	Pool *pgxpool.Pool
}

func NewMetricRepo(ctx context.Context, cfg *serverutils.ServerConfig) (*MetricRepo, error) {
	repo := &MetricRepo{}
	dbpool, err := pgxpool.Connect(ctx, cfg.DBDSN)
	if err != nil {
		return nil, err
	}
	repo.Pool = dbpool
	ct, err := repo.Pool.Exec(ctx, SQLCreateTable)
	if err != nil {
		return nil, err
	}
	log.Info().Str("initialize table", ct.String())
	if !cfg.Restore {
		_, err = repo.Pool.Exec(ctx, SQLTruncateTable)
		if err != nil {
			return nil, err
		}
	}
	return repo, nil
}

//этот метод не используется для варианта с ДБ
func (m *MetricRepo) DumpMetrics(ctx context.Context, cfg *serverutils.ServerConfig) error {
	return nil
}

func (m *MetricRepo) Close() {
	m.Pool.Close()
}

func (m *MetricRepo) Ping(ctx context.Context) error {
	return m.Pool.Ping(ctx)
}

func (m *MetricRepo) SaveMetric(ctx context.Context, metric metrics.Metrics) error {
	var oldValue string
	row := m.Pool.QueryRow(ctx, SQLSelectValueType, metric.ID)
	err := row.Scan(&oldValue)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		oldValue = metric.MType
	}
	if oldValue != metric.MType {
		return metrics.ErrUndefinedType
	}
	switch metric.MType {
	case "gauge":
		_, err := m.Pool.Exec(ctx, SQLInsertGaugeValue, metric.ID, metric.Value)
		if err != nil {
			return err
		}
	case "counter":
		_, err := m.Pool.Exec(ctx, SQLInsertCounterValue, metric.ID, metric.Delta)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MetricRepo) SaveListMetric(ctx context.Context, metricarray []metrics.Metrics) (int, error) {
	counter := 0
	tx, err := m.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	for _, metric := range metricarray {
		var oldValue string
		row := tx.QueryRow(ctx, SQLSelectValueType, metric.ID)
		err := row.Scan(&oldValue)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				log.Error().Err(err).Msg("failed get previous value")
				continue
			}
			oldValue = metric.MType
		}
		if oldValue != metric.MType {
			log.Error().Err(metrics.ErrUndefinedType).Msg("wrong metric type")
			continue
		}
		switch metric.MType {
		case "gauge":
			_, err := tx.Exec(ctx, SQLInsertGaugeValue, metric.ID, metric.Value)
			if err != nil {
				log.Error().Err(err).Msg("failed update gauge metric")
				continue
			}
		case "counter":
			_, err := tx.Exec(ctx, SQLInsertCounterValue, metric.ID, metric.Delta)
			if err != nil {
				log.Error().Err(err).Msg("failed update counter metric")
				continue
			}
		}
		counter++
	}
	if err := tx.Commit(ctx); err != nil {
		log.Error().Err(err).Msg("update drivers: unable to commit")
		return 0, err
	}
	return counter, nil
}

func (m *MetricRepo) ListMetrics(ctx context.Context) []string {
	var list []string
	rows, err := m.Pool.Query(ctx, SQLSelectAllValues)
	if err != nil {
		return list
	}
	defer rows.Close()
	for rows.Next() {
		var metric metrics.Metrics
		err = rows.Scan(&metric.ID, &metric.MType, &metric.Value, &metric.Delta)
		if err != nil {
			log.Error().Err(err).Msg("scan error for list metrics")
			continue
		}
		list = append(list, metric.ID+":"+metric.ValueString())
	}
	err = rows.Err()
	if err != nil {
		log.Error().Err(err).Msg("error in scan multiple values of metrics")
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i] < list[j]
	})
	return list
}

func (m *MetricRepo) GetValue(ctx context.Context, metricName string) (metrics.Metrics, error) {
	var metric metrics.Metrics
	row := m.Pool.QueryRow(ctx, SQLSelectValue, metricName)
	err := row.Scan(&metric.ID, &metric.MType, &metric.Value, &metric.Delta)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return metrics.Metrics{}, err
		}
		return metrics.Metrics{}, errors.New("metric not saved")
	}
	return metric, nil
}
