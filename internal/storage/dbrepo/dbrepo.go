// dbrepo - реализация интерфейса Repositorier с использованием Postgres.
package dbrepo

import (
	"context"
	"embed"
	"errors"
	"sort"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/rs/zerolog"
)

var log = zerolog.New(serverutils.LogConfig()).With().Timestamp().Str("component", "dbrepo").Logger()

// Файлы SQL хранятся в директории ./sql/
//
//go:embed sql/*.sql
var SQL embed.FS

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
	sqlBytes, err := SQL.ReadFile("sql/SQLCreateTable.sql")
	if err != nil {
		return nil, err
	}
	sqlQuery := string(sqlBytes)
	ct, err := repo.Pool.Exec(ctx, sqlQuery)
	if err != nil {
		return nil, err
	}
	log.Info().Str("initialize table", ct.String())
	if !cfg.Restore {
		sqlBytes, err = SQL.ReadFile("sql/SQLTruncateTable.sql")
		if err != nil {
			return nil, err
		}
		sqlQuery = string(sqlBytes)
		_, err = repo.Pool.Exec(ctx, sqlQuery)
		if err != nil {
			return nil, err
		}
	}
	return repo, nil
}

// этот метод не используется для варианта с ДБ
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
	sqlBytes, err := SQL.ReadFile("sql/SQLSelectValueType.sql")
	if err != nil {
		return err
	}
	sqlQuery := string(sqlBytes)
	row := m.Pool.QueryRow(ctx, sqlQuery, metric.ID)
	err = row.Scan(&oldValue)
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
		sqlBytes, err = SQL.ReadFile("sql/SQLInsertGaugeValue.sql")
		if err != nil {
			return err
		}
		sqlQuery = string(sqlBytes)
		_, err = m.Pool.Exec(ctx, sqlQuery, metric.ID, metric.Value)
		if err != nil {
			return err
		}
	case "counter":
		sqlBytes, err = SQL.ReadFile("sql/SQLInsertCounterValue.sql")
		if err != nil {
			return err
		}
		sqlQuery = string(sqlBytes)
		_, err := m.Pool.Exec(ctx, sqlQuery, metric.ID, metric.Delta)
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
		sqlBytes, err := SQL.ReadFile("sql/SQLSelectValueType.sql")
		if err != nil {
			return 0, err
		}
		sqlQuery := string(sqlBytes)
		row := tx.QueryRow(ctx, sqlQuery, metric.ID)
		err = row.Scan(&oldValue)
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
			sqlBytes, err := SQL.ReadFile("sql/SQLInsertGaugeValue.sql")
			if err != nil {
				return 0, err
			}
			sqlQuery := string(sqlBytes)
			_, err = tx.Exec(ctx, sqlQuery, metric.ID, metric.Value)
			if err != nil {
				log.Error().Err(err).Msg("failed update gauge metric")
				continue
			}
		case "counter":
			sqlBytes, err := SQL.ReadFile("sql/SQLInsertCounterValue.sql")
			if err != nil {
				return 0, err
			}
			sqlQuery := string(sqlBytes)
			_, err = tx.Exec(ctx, sqlQuery, metric.ID, metric.Delta)
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
	sqlBytes, err := SQL.ReadFile("sql/SQLSelectAllValues.sql")
	if err != nil {
		return nil
	}
	sqlQuery := string(sqlBytes)
	rows, err := m.Pool.Query(ctx, sqlQuery)
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
	sqlBytes, err := SQL.ReadFile("sql/SQLSelectValue.sql")
	if err != nil {
		return metrics.Metrics{}, err
	}
	sqlQuery := string(sqlBytes)
	row := m.Pool.QueryRow(ctx, sqlQuery, metricName)
	err = row.Scan(&metric.ID, &metric.MType, &metric.Value, &metric.Delta)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return metrics.Metrics{}, err
		}
		return metrics.Metrics{}, errors.New("metric not saved")
	}
	return metric, nil
}
