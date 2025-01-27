package main

import (
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
		if r.URL.Path != "/update/gauge/TestMetric/123.45" {
			t.Errorf("unexpected URL path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("expected Content-Type 'text/plain', got '%s'", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	agent := NewAgent(server.URL, 2*time.Second, 10*time.Second)

	agent.SendMetric("gauge", "TestMetric", 123.45)
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
