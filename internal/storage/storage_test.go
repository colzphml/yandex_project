package storage

import (
	"testing"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricRepo_GetValue(t *testing.T) {
	type args struct {
		metricName string
	}
	tests := []struct {
		name      string
		fields    MetricRepo
		args      args
		wantValue string
		wantType  string
		wantErr   bool
	}{
		{
			name: "Test #1: get Gauge value",
			fields: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(500.123),
					"Counter": metrics.Counter(200),
				}},
			args: args{
				metricName: "Gauge",
			},
			wantValue: "500.123",
			wantType:  "gauge",
			wantErr:   false,
		},
		{
			name: "Test #2: get Counter value",
			fields: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(500.123),
					"Counter": metrics.Counter(200),
				}},
			args: args{
				metricName: "Counter",
			},
			wantValue: "200",
			wantType:  "counter",
			wantErr:   false,
		},
		{
			name: "Test #2: get unknown metric",
			fields: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(500.123),
					"Counter": metrics.Counter(200),
				}},
			args: args{
				metricName: "ololo",
			},
			wantValue: "",
			wantType:  "",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MetricRepo{
				db: tt.fields.db,
			}
			mValue, err := m.GetValue(tt.args.metricName)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantValue, mValue.String())
				assert.Equal(t, tt.wantType, mValue.Type())
			}
		})
	}
}

func TestMetricRepo_SaveMetric(t *testing.T) {
	type args struct {
		metricName  string
		MetricValue metrics.MetricValue
	}
	tests := []struct {
		name     string
		fields   MetricRepo
		args     args
		wantRepo MetricRepo
		wantErr  bool
	}{
		{
			name: "Test #1: add Gauge value",
			fields: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(500.123),
					"Counter": metrics.Counter(200),
				}},
			args: args{
				metricName:  "Gauge",
				MetricValue: metrics.Gauge(300.123),
			},
			wantRepo: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(300.123),
					"Counter": metrics.Counter(200),
				}},
			wantErr: false,
		},
		{
			name: "Test #2: add Counter value",
			fields: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(500.123),
					"Counter": metrics.Counter(200),
				}},
			args: args{
				metricName:  "Counter",
				MetricValue: metrics.Counter(300),
			},
			wantRepo: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(500.123),
					"Counter": metrics.Counter(500),
				}},
			wantErr: false,
		},
		{
			name: "Test #3: add new metric",
			fields: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(500.123),
					"Counter": metrics.Counter(200),
				}},
			args: args{
				metricName:  "NewMetric",
				MetricValue: metrics.Counter(200),
			},
			wantRepo: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":     metrics.Gauge(500.123),
					"Counter":   metrics.Counter(200),
					"NewMetric": metrics.Counter(200),
				}},
			wantErr: false,
		},
		{
			name: "Test #4: add bad type",
			fields: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(500.123),
					"Counter": metrics.Counter(200),
				}},
			args: args{
				metricName:  "Gauge",
				MetricValue: metrics.Counter(200),
			},
			wantRepo: MetricRepo{
				db: map[string]metrics.MetricValue{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MetricRepo{
				db: tt.fields.db,
			}
			err := m.SaveMetric(tt.args.metricName, tt.args.MetricValue)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRepo, m)
			}
		})
	}
}

func TestMetricRepo_ListMetrics(t *testing.T) {
	tests := []struct {
		name   string
		fields MetricRepo
		want   []string
	}{
		{
			name: "Test #1: list metrics",
			fields: MetricRepo{
				db: map[string]metrics.MetricValue{
					"Gauge":   metrics.Gauge(500.123),
					"Counter": metrics.Counter(200),
				}},
			want: []string{"Gauge:500.123", "Counter:200"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MetricRepo{
				db: tt.fields.db,
			}
			result := m.ListMetrics()
			assert.ElementsMatch(t, tt.want, result)
		})
	}
}
