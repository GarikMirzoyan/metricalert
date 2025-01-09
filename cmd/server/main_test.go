package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUpdateHandler(t *testing.T) {
	storage := NewMemStorage()
	server := NewServer(storage)

	tests := []struct {
		name           string
		method         string
		url            string
		expectedStatus int
	}{
		{
			name:           "Valid Gauge Metric",
			method:         http.MethodPost,
			url:            "/update/gauge/Alloc/123.45",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid Counter Metric",
			method:         http.MethodPost,
			url:            "/update/counter/RequestCount/10",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid Method",
			method:         http.MethodGet,
			url:            "/update/gauge/Alloc/123.45",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing Metric Name",
			method:         http.MethodPost,
			url:            "/update/gauge//123.45",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid Metric Type",
			method:         http.MethodPost,
			url:            "/update/unknown/Alloc/123.45",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Gauge Value",
			method:         http.MethodPost,
			url:            "/update/gauge/Alloc/invalid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Counter Value",
			method:         http.MethodPost,
			url:            "/update/counter/RequestCount/invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)
			rec := httptest.NewRecorder()

			server.UpdateHandler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestGaugeUpdate(t *testing.T) {
	storage := NewMemStorage()
	server := NewServer(storage)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/TestMetric/123.45", nil)
	rec := httptest.NewRecorder()

	server.UpdateHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	value, exists := storage.gauges["TestMetric"]
	if !exists {
		t.Error("Gauge metric not found in storage")
	}
	if value.Value != 123.45 {
		t.Errorf("expected gauge value 123.45, got %f", value.Value)
	}
}

func TestCounterUpdate(t *testing.T) {
	storage := NewMemStorage()
	server := NewServer(storage)

	// First update
	req1 := httptest.NewRequest(http.MethodPost, "/update/counter/TestMetric/10", nil)
	rec1 := httptest.NewRecorder()
	server.UpdateHandler(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	// Second update
	req2 := httptest.NewRequest(http.MethodPost, "/update/counter/TestMetric/15", nil)
	rec2 := httptest.NewRecorder()
	server.UpdateHandler(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec2.Code)
	}

	value, exists := storage.counters["TestMetric"]
	if !exists {
		t.Error("Counter metric not found in storage")
	}
	if value.Value != 25 {
		t.Errorf("expected counter value 25, got %d", value.Value)
	}
}
