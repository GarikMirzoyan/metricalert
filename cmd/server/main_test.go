package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

func TestUpdateHandler(t *testing.T) {
	// Создаем логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync() // Ensure to flush any buffered log entries

	storage := NewMemStorage()
	server := NewServer(storage, logger)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", server.UpdateHandler)

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

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestUpdateHandlerJson(t *testing.T) {
	// Создаем логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync() // Ensure to flush any buffered log entries

	storage := NewMemStorage()
	server := NewServer(storage, logger)

	r := chi.NewRouter()
	r.Post("/update/", server.UpdateHandler)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
	}{
		{
			name:           "Valid Gauge Metric",
			method:         http.MethodPost,
			body:           `{"id":"Alloc","type":"gauge","value":123.45}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid Counter Metric",
			method:         http.MethodPost,
			body:           `{"id":"RequestCount","type":"counter","delta":10}`,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid Method",
			method:         http.MethodGet,
			body:           `{"id":"Alloc","type":"gauge","value":123.45}`,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Missing Metric Name",
			method:         http.MethodPost,
			body:           `{"type":"gauge","value":123.45}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Metric Type",
			method:         http.MethodPost,
			body:           `{"id":"Alloc","type":"unknown","value":123.45}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Gauge Value",
			method:         http.MethodPost,
			body:           `{"id":"Alloc","type":"gauge","value":"invalid"}`,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid Counter Value",
			method:         http.MethodPost,
			body:           `{"id":"RequestCount","type":"counter","delta":"invalid"}`,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/update/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}

func TestGaugeUpdate(t *testing.T) {
	// Создаем логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync() // Ensure to flush any buffered log entries

	storage := NewMemStorage()
	server := NewServer(storage, logger)

	r := chi.NewRouter()
	r.Post("/update/", server.UpdateHandler)

	req := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(`{"id":"TestMetric","type":"gauge","value":123.45}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

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
	// Создаем логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync() // Ensure to flush any buffered log entries

	storage := NewMemStorage()
	server := NewServer(storage, logger)

	r := chi.NewRouter()
	r.Post("/update/", server.UpdateHandler)

	// First update
	req1 := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(`{"id":"TestMetric","type":"counter","delta":10}`))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec1.Code)
	}

	// Second update
	req2 := httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(`{"id":"TestMetric","type":"counter","delta":15}`))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

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

func normalizeHTML(input string) string {
	return strings.Join(strings.Fields(input), "")
}

func TestRootHandler(t *testing.T) {
	// Создаем логгер
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync() // Ensure to flush any buffered log entries

	storage := NewMemStorage()
	server := NewServer(storage, logger)

	r := chi.NewRouter()
	r.Get("/", server.RootHandler)

	// Update some metrics
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(`{"id":"TestGauge","type":"gauge","value":1.23}`)))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/update/", strings.NewReader(`{"id":"TestCounter","type":"counter","delta":5}`)))

	// Now test root handler
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	expectedBody := "<!DOCTYPE html>"
	actualBody := normalizeHTML(rec.Body.String())
	normalizedExpectedBody := normalizeHTML(expectedBody)

	if !strings.HasPrefix(actualBody, normalizedExpectedBody) {
		t.Errorf("expected body to start with '%s', but got '%s'", normalizedExpectedBody, actualBody)
	}
}

func TestGetValueHandlerPost(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	storage := NewMemStorage()
	server := NewServer(storage, logger)

	r := chi.NewRouter()
	r.Post("/getvalue/", server.GetValueHandlerPost)

	storage.UpdateGauge("Alloc", 123.45)
	storage.UpdateCounter("RequestCount", 100)

	tests := []struct {
		name           string
		method         string
		body           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid Gauge Metric",
			method:         http.MethodPost,
			body:           `{"id":"Alloc","type":"gauge"}`,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"Alloc","type":"gauge","value":123.45}`,
		},
		{
			name:           "Valid Counter Metric",
			method:         http.MethodPost,
			body:           `{"id":"RequestCount","type":"counter"}`,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"id":"RequestCount","type":"counter","delta":100}`,
		},
		{
			name:           "Metric Not Found",
			method:         http.MethodPost,
			body:           `{"id":"NonExistent","type":"gauge"}`,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `Metric not found`,
		},
		{
			name:           "Invalid Metric Type",
			method:         http.MethodPost,
			body:           `{"id":"Alloc","type":"unknown"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `Invalid metric type`,
		},
		{
			name:           "Missing Metric Type",
			method:         http.MethodPost,
			body:           `{"id":"Alloc"}`,
			expectedStatus: http.StatusNotFound,
			expectedBody:   `Metric name not provided`,
		},
		{
			name:           "Invalid JSON",
			method:         http.MethodPost,
			body:           `{"id":"Alloc","type"}`,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `Invalid JSON`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/getvalue/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if strings.TrimSpace(rec.Body.String()) != tt.expectedBody {
				t.Errorf("expected body %s, got %s", tt.expectedBody, rec.Body.String())
			}
		})
	}
}

// Helper function to check if a string contains a substring
func stringContains(str, substr string) bool {
	return len(str) >= len(substr) && str[:len(substr)] == substr
}
