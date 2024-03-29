package metricsserver

import (
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/colzphml/yandex_project/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToMetric(t *testing.T) {
	type args struct {
		metricName  string
		metricType  string
		metricValue string
	}
	tests := []struct {
		name      string
		args      args
		wantID    string
		wantType  string
		wantDelta int64
		wantValue float64
		wantErr   bool
	}{
		{
			name: "Test #1: convert to Gauge",
			args: args{
				metricName:  "test",
				metricType:  "gauge",
				metricValue: "7.77",
			},
			wantID:    "test",
			wantType:  "gauge",
			wantValue: 7.77,
			wantErr:   false,
		},
		{
			name: "Test #2: convert to Counter",
			args: args{
				metricName:  "test",
				metricType:  "counter",
				metricValue: "777",
			},
			wantID:    "test",
			wantType:  "counter",
			wantDelta: 777,
			wantErr:   false,
		},
		{
			name: "Test #3: convert to another",
			args: args{
				metricName:  "test",
				metricType:  "another",
				metricValue: "777",
			},
			wantErr: true,
		},
		{
			name: "Test #3: convert wrong type",
			args: args{
				metricName:  "test",
				metricType:  "counter",
				metricValue: "7.77",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertToMetric(tt.args.metricName, tt.args.metricType, tt.args.metricValue)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, got.ID)
				assert.Equal(t, tt.wantType, got.MType)
				switch got.MType {
				case "gauge":
					assert.Equal(t, tt.wantValue, *got.Value)
				case "counter":
					assert.Equal(t, tt.wantDelta, *got.Delta)
				}
			}
		})
	}
}

func TestNewValue(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		mtypeOld  string
		mtypeNew  string
		wantID    string
		wantType  string
		newDelta  int64
		oldDelta  int64
		wantDelta int64
		newValue  float64
		oldValue  float64
		wantValue float64
		wantErr   bool
	}{
		{
			name:      "Test #1: new value of Gauge",
			id:        "test",
			mtypeOld:  "gauge",
			mtypeNew:  "gauge",
			oldValue:  7.77,
			newValue:  8.88,
			wantID:    "test",
			wantType:  "gauge",
			wantValue: 8.88,
			wantErr:   false,
		},
		{
			name:      "Test #2: new value of Counter",
			id:        "test",
			mtypeOld:  "counter",
			mtypeNew:  "counter",
			oldDelta:  100,
			newDelta:  254,
			wantID:    "test",
			wantType:  "counter",
			wantDelta: 354,
			wantErr:   false,
		},
		{
			name:     "Test #3: different types",
			id:       "test",
			mtypeOld: "counter",
			mtypeNew: "gauge",
			wantErr:  true,
		},
		{
			name:     "Test #3: another type",
			id:       "test",
			mtypeOld: "another",
			mtypeNew: "another",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldMetric := metrics.Metrics{
				ID:    tt.id,
				MType: tt.mtypeOld,
				Value: &tt.oldValue,
				Delta: &tt.oldDelta,
			}
			newMetric := metrics.Metrics{
				ID:    tt.id,
				MType: tt.mtypeNew,
				Value: &tt.newValue,
				Delta: &tt.newDelta,
			}
			got, err := NewValue(oldMetric, newMetric)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, got.ID)
				assert.Equal(t, tt.wantType, got.MType)
				switch got.MType {
				case "gauge":
					assert.Equal(t, tt.wantValue, *got.Value)
				case "counter":
					assert.Equal(t, tt.wantDelta, *got.Delta)
				}
			}
		})
	}
}

func BenchmarkConvertToMetric(b *testing.B) {
	r := []string{
		"gauge",
		"counter",
	}
	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		//почему то не работает запуск таймера, программа зависает...
		//UPD похоже из-за частого запуска таймера...
		//b.StopTimer()
		mtype := r[rand.Intn(2)]
		switch mtype {
		case "gauge":
			value := rand.Float64()
			valuestr := strconv.FormatFloat(value, 'g', -1, 64)
			//b.StartTimer()
			ConvertToMetric("test", mtype, valuestr)
		case "counter":
			value := rand.Int63()
			valuestr := strconv.FormatInt(value, 10)
			//b.StartTimer()
			ConvertToMetric("test", mtype, valuestr)
		}
	}
}

func BenchmarkNewValue(b *testing.B) {
	oldValue := metrics.Metrics{
		ID: "test",
	}
	newValue := metrics.Metrics{
		ID: "test",
	}
	rand.Seed(time.Now().UnixNano())
	valueOldFloat := rand.Float64()
	oldValue.MType = "gauge"
	oldValue.Value = &valueOldFloat
	valueNewFloat := rand.Float64()
	newValue.MType = "gauge"
	newValue.Value = &valueNewFloat
	b.ResetTimer()
	b.Run("gauge", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewValue(oldValue, newValue)
		}
	})
	valueOldInt := rand.Int63()
	oldValue.MType = "counter"
	oldValue.Delta = &valueOldInt
	valueNewInt := rand.Int63()
	newValue.MType = "counter"
	newValue.Delta = &valueNewInt
	b.ResetTimer()
	b.Run("counter", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			NewValue(oldValue, newValue)
		}
	})
}
