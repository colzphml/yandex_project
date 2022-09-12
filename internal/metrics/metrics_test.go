package metrics

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_ValueString(t *testing.T) {
	type fields struct {
		ID    string
		MType string
	}
	tests := []struct {
		name   string
		fields fields
		Value  float64
		Delta  int64
		want   string
	}{
		{
			name: "Test #1: get ValueString for Gauge",
			fields: fields{
				ID:    "test",
				MType: "gauge",
			},
			Value: 6.97,
			want:  "6.97",
		},
		{
			name: "Test #2: get ValueString for Counter",
			fields: fields{
				ID:    "test",
				MType: "counter",
			},
			Delta: 1812,
			want:  "1812",
		},
		{
			name: "Test #3: get ValueString for another type",
			fields: fields{
				ID:    "test",
				MType: "another",
			},
			Delta: 1812,
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metrics{
				ID:    tt.fields.ID,
				MType: tt.fields.MType,
				Delta: &tt.Delta,
				Value: &tt.Value,
			}
			got := m.ValueString()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMetrics_CalculateHash(t *testing.T) {
	type fields struct {
		ID    string
		MType string
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		value   float64
		delta   int64
		want    []byte
		wantErr bool
	}{
		{
			name: "Test #1: get hash for Gauge",
			fields: fields{
				ID:    "test",
				MType: "gauge",
			},
			args:    args{key: "test"},
			value:   7.77,
			want:    []byte{135, 211, 87, 250, 49, 24, 253, 48, 31, 164, 25, 67, 130, 222, 41, 221, 118, 114, 35, 94, 184, 196, 89, 13, 59, 124, 75, 130, 178, 94, 156, 230},
			wantErr: false,
		},
		{
			name: "Test #2: get hash for Counter",
			fields: fields{
				ID:    "test",
				MType: "counter",
			},
			args:    args{key: "test"},
			delta:   777,
			want:    []byte{4, 103, 211, 183, 9, 120, 174, 158, 255, 102, 18, 50, 254, 11, 46, 61, 7, 255, 221, 194, 115, 138, 53, 60, 81, 86, 96, 198, 210, 72, 92, 65},
			wantErr: false,
		},
		{
			name: "Test #3: get hash for another type",
			fields: fields{
				ID:    "test",
				MType: "another",
			},
			args:    args{key: "test"},
			want:    []byte{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metrics{
				ID:    tt.fields.ID,
				MType: tt.fields.MType,
				Value: &tt.value,
				Delta: &tt.delta,
			}
			got, err := m.CalculateHash(tt.args.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMetrics_FillHash(t *testing.T) {
	type fields struct {
		ID    string
		MType string
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		value   float64
		delta   int64
		want    string
		wantErr bool
	}{
		{
			name: "Test #1: fill hash for Gauge",
			fields: fields{
				ID:    "test",
				MType: "gauge",
			},
			args:    args{key: "test"},
			value:   7.77,
			want:    "87d357fa3118fd301fa4194382de29dd7672235eb8c4590d3b7c4b82b25e9ce6",
			wantErr: false,
		},
		{
			name: "Test #2: fill hash for Counter",
			fields: fields{
				ID:    "test",
				MType: "counter",
			},
			args:    args{key: "test"},
			delta:   777,
			want:    "0467d3b70978ae9eff661232fe0b2e3d07ffddc2738a353c515660c6d2485c41",
			wantErr: false,
		},
		{
			name: "Test #3: fill hash for another type",
			fields: fields{
				ID:    "test",
				MType: "another",
			},
			args:    args{key: "test"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metrics{
				ID:    tt.fields.ID,
				MType: tt.fields.MType,
				Value: &tt.value,
				Delta: &tt.delta,
			}
			err := m.FillHash(tt.args.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, m.Hash)
			}
		})
	}
}

func TestMetrics_CompareHash(t *testing.T) {
	type fields struct {
		ID    string
		MType string
		Hash  string
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		value   float64
		delta   int64
		want    bool
		wantErr bool
	}{
		{
			name: "Test #1: compare hash for Gauge",
			fields: fields{
				ID:    "test",
				MType: "gauge",
				Hash:  "87d357fa3118fd301fa4194382de29dd7672235eb8c4590d3b7c4b82b25e9ce6",
			},
			args:    args{key: "test"},
			value:   7.77,
			want:    true,
			wantErr: false,
		},
		{
			name: "Test #2: compare hash for Counter",
			fields: fields{
				ID:    "test",
				MType: "counter",
				Hash:  "0467d3b70978ae9eff661232fe0b2e3d07ffddc2738a353c515660c6d2485c41",
			},
			args:    args{key: "test"},
			delta:   777,
			want:    true,
			wantErr: false,
		},
		{
			name: "Test #3: compare hash for another type",
			fields: fields{
				ID:    "test",
				MType: "another",
			},
			args:    args{key: "test"},
			want:    true,
			wantErr: true,
		},
		{
			name: "Test #4: wrong hash for Gauge",
			fields: fields{
				ID:    "test",
				MType: "gauge",
				Hash:  "87d357fa3118fd301fa4194382de29dd7672235eb8c4590d3b7c4b82b25e9ce7",
			},
			args:    args{key: "test"},
			value:   7.77,
			want:    false,
			wantErr: false,
		},
		{
			name: "Test #5: wrong hash for Counter",
			fields: fields{
				ID:    "test",
				MType: "counter",
				Hash:  "0467d3b70978ae9eff661232fe0b2e3d07ffddc2738a353c515660c6d2485c42",
			},
			args:    args{key: "test"},
			delta:   777,
			want:    false,
			wantErr: false,
		},
		{
			name: "Test #6: empty key",
			fields: fields{
				ID:    "test",
				MType: "counter",
				Hash:  "0467d3b70978ae9eff661232fe0b2e3d07ffddc2738a353c515660c6d2485c42",
			},
			args:    args{key: ""},
			delta:   777,
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metrics{
				ID:    tt.fields.ID,
				MType: tt.fields.MType,
				Value: &tt.value,
				Delta: &tt.delta,
				Hash:  tt.fields.Hash,
			}
			got, err := m.CompareHash(tt.args.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func BenchmarkFillHash(b *testing.B) {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	buf := make([]rune, 10)
	for i := range buf {
		buf[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	str := string(buf)
	value := 777.77
	metric := Metrics{
		ID:    "test",
		MType: "gauge",
		Value: &value,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metric.FillHash(str)
	}
}
