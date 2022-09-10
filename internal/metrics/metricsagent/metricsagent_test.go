package metricsagent

import (
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

func BenchmarkGetRuntimeMetric(b *testing.B) {
	var runtimeState runtime.MemStats
	b.Run("Alloc", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			getRuntimeMetric(&runtimeState, "Alloc", "gauge")
		}
	})
	b.Run("NextGC", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			getRuntimeMetric(&runtimeState, "NextGC", "gauge")
		}
	})
	b.Run("HeapInuse", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			getRuntimeMetric(&runtimeState, "HeapInuse", "gauge")
		}
	})

}

func BenchmarkReadRuntimeMetrics(b *testing.B) {
	r := NewRepo()
	metricsDescr := map[string]string{
		"Alloc":     "gauge",
		"NextGC":    "gauge",
		"HeapInuse": "gauge",
	}
	var runtimeState runtime.MemStats
	rand.Seed(time.Now().UnixNano())
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReadRuntimeMetrics(r, metricsDescr, &runtimeState, rand.Int63()%1000)
	}
}

func BenchmarkReadSystemMetrics(b *testing.B) {
	r := NewRepo()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ReadSystemMetrics(r)
	}
}

type MetricsAgentSuite struct {
	suite.Suite
	runtime      *runtime.MemStats
	repo         *MetricRepo
	metricsDescr map[string]string
}

func (suite *MetricsAgentSuite) SetupTest() {
	var runtimeState runtime.MemStats
	suite.runtime = &runtimeState
	suite.metricsDescr = map[string]string{
		"Alloc":     "gauge",
		"NextGC":    "gauge",
		"HeapInuse": "gauge",
	}
	suite.repo = NewRepo()
}

func (suite *MetricsAgentSuite) TestgetRuntimeMetricAlloc() {
	m, err := getRuntimeMetric(suite.runtime, "Alloc", suite.metricsDescr["Alloc"])
	suite.NoError(err)
	suite.Equal("Alloc", m.ID)
	suite.Equal(m.MType, suite.metricsDescr["Alloc"])
	suite.NotNil(m.Value)
}

func (suite *MetricsAgentSuite) TestgetRuntimeMetricUnknown() {
	_, err := getRuntimeMetric(suite.runtime, "another", "gauge")
	suite.Error(err)
}

func (suite *MetricsAgentSuite) TestgetRuntimeMetricWrongType() {
	_, err := getRuntimeMetric(suite.runtime, "Alloc", "another")
	suite.Error(err)
}

func (suite *MetricsAgentSuite) TestReadRuntimeMetrics() {
	ReadRuntimeMetrics(suite.repo, suite.metricsDescr, suite.runtime, 1)
	suite.Equal(5, len(suite.repo.db))
}

func (suite *MetricsAgentSuite) TestReadRuntimeMetricsInvalidName() {
	suite.metricsDescr["another"] = "gauge"
	ReadRuntimeMetrics(suite.repo, suite.metricsDescr, suite.runtime, 1)
	suite.Equal(5, len(suite.repo.db))
}

func (suite *MetricsAgentSuite) TestReadSystemMetrics() {
	ReadSystemMetrics(suite.repo)
	suite.GreaterOrEqual(len(suite.repo.db), 3)
	_, ok := suite.repo.db["TotalMemory"]
	suite.Equal(true, ok)
	_, ok = suite.repo.db["FreeMemory"]
	suite.Equal(true, ok)
	_, ok = suite.repo.db["CPUutilization1"]
	suite.Equal(true, ok)
}

func TestExampleTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsAgentSuite))
}
