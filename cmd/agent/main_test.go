package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCollectMetrics(t *testing.T) {
	agent := NewAgent("http://localhost:8080", 2*time.Second, 10*time.Second)

	metrics := agent.CollectMetrics()

	if len(metrics) == 0 {
		t.Errorf("expected metrics to be collected, but got none")
	}

	if _, ok := metrics["Alloc"]; !ok {
		t.Errorf("expected metric 'Alloc' to be collected")
	}

	if _, ok := metrics["RandomValue"]; !ok {
		t.Errorf("expected metric 'RandomValue' to be collected")
	}
}

func TestSendMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}

		// Проверяем заголовки для gzip
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("expected Content-Encoding 'gzip', got '%s'", r.Header.Get("Content-Encoding"))
		}

		// Декодируем тело сжатое gzip
		gzipReader, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()

		// Читаем данные из gzip
		body, err := io.ReadAll(gzipReader)
		if err != nil {
			t.Errorf("failed to read gzip body: %v", err)
		}

		// Декодируем JSON
		var metric Metrics
		if err := json.Unmarshal(body, &metric); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}

		// Проверяем данные метрики
		if metric.ID != "TestMetric" {
			t.Errorf("expected metric ID 'TestMetric', got %s", metric.ID)
		}
		if metric.MType != "gauge" {
			t.Errorf("expected metric type 'gauge', got %s", metric.MType)
		}
		if *metric.Value != 123.45 {
			t.Errorf("expected metric value 123.45, got %f", *metric.Value)
		}

		// Проверяем Content-Type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Ответ на успешную обработку
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgent(server.URL, 2*time.Second, 10*time.Second)

	value := 123.45
	metric := Metrics{
		ID:    "TestMetric",
		MType: "gauge",
		Value: &value,
	}

	agent.SendMetric(metric)
}

func TestAgentRun(t *testing.T) {
	requests := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgent(server.URL, 1*time.Second, 3*time.Second)

	go agent.Run()

	// Allow some time for metrics to be collected and sent
	time.Sleep(5 * time.Second)

	if requests == 0 {
		t.Errorf("expected at least one request to be sent, but got %d", requests)
	}
}

func TestPollAndReportIntervals(t *testing.T) {
	pollInterval := 500 * time.Millisecond
	reportInterval := 1 * time.Second

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgent(server.URL, pollInterval, reportInterval)

	go agent.Run()

	// Allow some time for metrics collection and reporting
	time.Sleep(3 * reportInterval)

	if agent.pollCount < 3 {
		t.Errorf("expected pollCount to be at least 3, got %d", agent.pollCount)
	}
}
