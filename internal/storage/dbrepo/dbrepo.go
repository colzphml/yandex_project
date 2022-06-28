package dbrepo

import (
	"context"
	"errors"
	"log"
	"sort"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/colzphml/yandex_project/internal/serverutils"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrNoDB = errors.New("connection to db was not created")
)

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

func NewMetricRepo(cfg *serverutils.ServerConfig) (*MetricRepo, error) {
	repo := &MetricRepo{}
	dbpool, err := pgxpool.Connect(context.Background(), cfg.DBDSN)
	if err != nil {
		return nil, err
	}
	repo.Pool = dbpool
	ct, err := repo.Pool.Exec(context.Background(), SQLCreateTable)
	if err != nil {
		return nil, err
	}
	log.Printf("initialize table: %s\n", ct.String())
	if !cfg.Restore {
		_, err = repo.Pool.Exec(context.Background(), SQLTruncateTable)
		if err != nil {
			return nil, err
		}
	}
	return repo, nil
}

//этот метод не используется для варианта с ДБ
func (m *MetricRepo) DumpMetrics(cfg *serverutils.ServerConfig) error {
	return nil
}

func (m *MetricRepo) Close() {
	m.Pool.Close()
}

func (m *MetricRepo) Ping() error {
	return m.Pool.Ping(context.Background())
}

func (m *MetricRepo) SaveMetric(metric metrics.Metrics) error {
	var oldValue string
	row := m.Pool.QueryRow(context.Background(), SQLSelectValueType, metric.ID)
	err := row.Scan(&oldValue)
	if err != nil {
		if err != pgx.ErrNoRows {
			return err
		}
		oldValue = metric.MType
	}
	if oldValue != metric.MType {
		return metrics.ErrUndefinedType
	}
	switch metric.MType {
	case "gauge":
		_, err := m.Pool.Exec(context.Background(), SQLInsertGaugeValue, metric.ID, metric.Value)
		if err != nil {
			return err
		}
	case "counter":
		_, err := m.Pool.Exec(context.Background(), SQLInsertCounterValue, metric.ID, metric.Delta)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *MetricRepo) ListMetrics() []string {
	var list []string
	rows, err := m.Pool.Query(context.Background(), SQLSelectAllValues)
	if err != nil {
		return list
	}
	defer rows.Close()
	for rows.Next() {
		var metric metrics.Metrics
		err = rows.Scan(&metric.ID, &metric.MType, &metric.Value, &metric.Delta)
		if err != nil {
			log.Println("scan err: " + err.Error())
			continue
		}
		list = append(list, metric.ID+":"+metric.ValueString())
	}
	err = rows.Err()
	if err != nil {
		log.Println(err)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i] < list[j]
	})
	return list
}

func (m *MetricRepo) GetValue(metricName string) (metrics.Metrics, error) {
	var metric metrics.Metrics
	row := m.Pool.QueryRow(context.Background(), SQLSelectValue, metricName)
	err := row.Scan(&metric.ID, &metric.MType, &metric.Value, &metric.Delta)
	if err != nil {
		if err != pgx.ErrNoRows {
			return metrics.Metrics{}, err
		}
		return metrics.Metrics{}, errors.New("metric not saved")
	}
	return metric, nil
}
