package metrics

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	agentConfig "github.com/GarikMirzoyan/metricalert/internal/agent/config"
	serverConfig "github.com/GarikMirzoyan/metricalert/internal/server/config"
	"github.com/stretchr/testify/assert"
)

func TestNewMemStorage(t *testing.T) {
	ms := NewMemStorage()
	assert.NotNil(t, ms)
	assert.Empty(t, ms.gauges)
	assert.Empty(t, ms.counters)
}

func TestUpdateGauge(t *testing.T) {
	ms := NewMemStorage()
	ms.UpdateGauge("test_gauge", 10.5)

	gauge, exists := ms.GetGauge("test_gauge")
	assert.True(t, exists)
	assert.Equal(t, 10.5, gauge.Value)
}

func TestUpdateCounter(t *testing.T) {
	ms := NewMemStorage()
	ms.UpdateCounter("test_counter", 5)
	ms.UpdateCounter("test_counter", 10)

	counter, exists := ms.GetCounter("test_counter")
	assert.True(t, exists)
	assert.Equal(t, int64(15), counter.Value)
}

func TestSaveMetricsToFile(t *testing.T) {
	config := serverConfig.Config{
		FileStoragePath: "data/test_output_metrics.json",
	}
	ms := NewMemStorage()

	ms.UpdateGauge("test_gauge", 10.5)

	err := ms.SaveMetricsToFile(config)
	assert.NoError(t, err)

}

func TestLoadMetricsFromFile(t *testing.T) {
	config := serverConfig.Config{
		Restore:         true,
		FileStoragePath: "data/test_output_metrics.json",
	}
	ms := NewMemStorage()

	err := ms.LoadMetricsFromFile(config)
	assert.NoError(t, err)
}

func TestUpdateMetricsInvalidType(t *testing.T) {
	ms := NewMemStorage()
	err := ms.UpdateMetrics("invalid_type", "10", "test_gauge")
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidMetricType, err)
}

func TestUpdateMetricsFromJSON(t *testing.T) {
	request := &http.Request{
		Body:   mockBody(`{"id": "test_gauge", "type": "gauge", "value": 15.5}`),
		Header: make(http.Header),
	}
	ms := NewMemStorage()

	metric, err := ms.UpdateMetricsFromJSON(request)
	assert.NoError(t, err)
	assert.Equal(t, "test_gauge", metric.ID)
	assert.Equal(t, "gauge", metric.MType)
	assert.Equal(t, 15.5, *metric.Value)
}

func TestGetMetricsFromJSON(t *testing.T) {
	request := &http.Request{
		Body:   mockBody(`{"id": "test_gauge", "type": "gauge"}`),
		Header: make(http.Header),
	}
	ms := NewMemStorage()
	ms.UpdateGauge("test_gauge", 10.5)

	metric, err := ms.GetMetricsFromJSON(request)
	assert.NoError(t, err)
	assert.Equal(t, "test_gauge", metric.ID)
	assert.Equal(t, "gauge", metric.MType)
	assert.Equal(t, 10.5, *metric.Value)
}

func TestGetMetricValueGauge(t *testing.T) {
	ms := NewMemStorage()
	ms.UpdateGauge("test_gauge", 10.5)

	result, err := ms.GetMetricValue("gauge", "test_gauge")
	assert.NoError(t, err)
	assert.Equal(t, "10.5", result)
}

func TestGetMetricValueCounter(t *testing.T) {
	ms := NewMemStorage()
	ms.UpdateCounter("test_counter", 15)

	result, err := ms.GetMetricValue("counter", "test_counter")
	assert.NoError(t, err)
	assert.Equal(t, "15", result)
}

func TestSendMetric(t *testing.T) {
	config := agentConfig.Config{
		Address: "http://localhost:8080",
	}
	metric := Metrics{
		ID:    "test_gauge",
		MType: string(GaugeName),
		Value: float64Ptr(10.5),
	}

	assert.NotPanics(t, func() {
		SendMetric(metric, config)
	})
}

func mockBody(body string) io.ReadCloser {
	return &mockReadCloser{bytes.NewReader([]byte(body))}
}

type mockReadCloser struct {
	*bytes.Reader
}

func (m *mockReadCloser) Close() error {
	return nil
}

func float64Ptr(val float64) *float64 {
	return &val
}

func TestGetAllMetrics(t *testing.T) {
	ms := NewMemStorage()
	ms.UpdateGauge("gauge_1", 10.5)
	ms.UpdateCounter("counter_1", 15)

	gauges, counters := ms.GetAllMetrics()
	assert.Equal(t, 1, len(gauges))
	assert.Equal(t, 1, len(counters))
	assert.Equal(t, 10.5, gauges["gauge_1"])
	assert.Equal(t, int64(15), counters["counter_1"])
}

func TestUpdateMetricsFromJSONInvalidJson(t *testing.T) {
	request := &http.Request{
		Body:   mockBody(`{"id": "test_gauge", "type": "gauge", "value": "invalid"}`),
		Header: make(http.Header),
	}
	ms := NewMemStorage()

	_, err := ms.UpdateMetricsFromJSON(request)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidJSON, err)
}
