package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValue(t *testing.T) {
	type args struct {
		oldValue MetricValue
		newValue MetricValue
	}
	tests := []struct {
		name    string
		args    args
		want    MetricValue
		wantErr bool
	}{
		{
			name: "Test #1: new gauge value",
			args: args{
				oldValue: Gauge(500.123),
				newValue: Gauge(100.321),
			},
			want:    Gauge(100.321),
			wantErr: false,
		},
		{
			name: "Test #2: new counter value",
			args: args{
				oldValue: Counter(100),
				newValue: Counter(200),
			},
			want:    Counter(300),
			wantErr: false,
		},
		{
			name: "Test #3: another type #1",
			args: args{
				oldValue: Counter(100),
				newValue: Gauge(200),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test #4: another type #2",
			args: args{
				oldValue: Gauge(100),
				newValue: Counter(200),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewValue(tt.args.oldValue, tt.args.newValue)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestConvertToMetric(t *testing.T) {
	type args struct {
		metricType  string
		metricValue string
	}
	tests := []struct {
		name    string
		args    args
		want    MetricValue
		wantErr bool
	}{
		{
			name: "Test #1: get MetricValue for Gauge",
			args: args{
				metricType:  "gauge",
				metricValue: "500.123",
			},
			want:    Gauge(500.123),
			wantErr: false,
		},
		{
			name: "Test #2: get MetricValue for Counter",
			args: args{
				metricType:  "counter",
				metricValue: "500",
			},
			want:    Counter(500),
			wantErr: false,
		},
		{
			name: "Test #3: counter to gauge",
			args: args{
				metricType:  "gauge",
				metricValue: "500",
			},
			want:    Gauge(500),
			wantErr: false,
		},
		{
			name: "Test #4: gauge to counter",
			args: args{
				metricType:  "counter",
				metricValue: "500.123",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Test #5: unknown type",
			args: args{
				metricType:  "ololo",
				metricValue: "500.123",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToMetric(tt.args.metricType, tt.args.metricValue)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
