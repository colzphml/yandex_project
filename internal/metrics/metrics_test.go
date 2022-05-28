package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValue(t *testing.T) {
	type args struct {
		oldValue   MetricValue
		metricName string
		newValue   MetricValue
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
				oldValue:   Gauge(500.123),
				metricName: "Gauge",
				newValue:   Gauge(100.321),
			},
			want:    Gauge(100.321),
			wantErr: false,
		},
		{
			name: "Test #2: new counter value",
			args: args{
				oldValue:   Counter(100),
				metricName: "Counter",
				newValue:   Counter(200),
			},
			want:    Counter(300),
			wantErr: false,
		},
		{
			name: "Test #3: another type #1",
			args: args{
				oldValue:   Counter(100),
				metricName: "Counter",
				newValue:   Gauge(200),
			},
			want:    struct{}{},
			wantErr: true,
		},
		{
			name: "Test #4: another type #2",
			args: args{
				oldValue:   Gauge(100),
				metricName: "Counter",
				newValue:   Counter(200),
			},
			want:    struct{}{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewValue(tt.args.oldValue, tt.args.metricName, tt.args.newValue)
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
			want:    struct{}{},
			wantErr: true,
		},
		{
			name: "Test #5: unknown type",
			args: args{
				metricType:  "ololo",
				metricValue: "500.123",
			},
			want:    struct{}{},
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

func TestMetricType(t *testing.T) {
	type args struct {
		a MetricValue
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test #1: get Gauge type",
			args: args{
				a: Gauge(500),
			},
			want: "gauge",
		},
		{
			name: "Test #2: get Counter type",
			args: args{
				a: Counter(500),
			},
			want: "counter",
		},
		{
			name: "Test #3: get string type",
			args: args{
				a: "test",
			},
			want: "",
		},
		{
			name: "Test #4: get int type",
			args: args{
				a: int(10),
			},
			want: "",
		},
		{
			name: "Test #4: get float type",
			args: args{
				a: float64(10.01),
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MetricType(tt.args.a)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValueToString(t *testing.T) {
	type args struct {
		a MetricValue
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test #1: get Gauge value",
			args: args{
				a: Gauge(500.123),
			},
			want: "500.123",
		},
		{
			name: "Test #2: get Counter value",
			args: args{
				a: Counter(500),
			},
			want: "500",
		},
		{
			name: "Test #3: get String value",
			args: args{
				a: "500",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValueToString(tt.args.a)
			assert.Equal(t, tt.want, got)
		})
	}
}
