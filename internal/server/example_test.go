package server_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	dto "github.com/GarikMirzoyan/metricalert/internal/DTO"
	"github.com/GarikMirzoyan/metricalert/internal/handlers"
	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	srv "github.com/GarikMirzoyan/metricalert/internal/server"
	"github.com/GarikMirzoyan/metricalert/internal/server/config"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

// Example demonstrating JSON update and read endpoints using in-memory storage.
func Example_updateAndGetJSON() {
	r := chi.NewRouter()
	logger, _ := zap.NewDevelopment()
	cfg := config.Config{Key: ""}

	srv.SetMiddlewares(r, logger, cfg)

	storage := metrics.NewMemStorage()
	h := handlers.NewHandlers(storage)
	srv.SetMetricRoutes(r, h)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Update gauge via JSON
	v := 42.5
	body, _ := json.Marshal(dto.Metrics{ID: "temperature", MType: "gauge", Value: &v})
	resp, err := http.Post(ts.URL+"/update/", "application/json", bytes.NewReader(body))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Read back via JSON
	reqBody, _ := json.Marshal(dto.Metrics{ID: "temperature", MType: "gauge"})
	resp2, err := http.Post(ts.URL+"/value/", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return
	}
	defer resp2.Body.Close()
	b, _ := io.ReadAll(resp2.Body)
	fmt.Println(string(bytes.TrimSpace(b)))
	// Output:
	// {"id":"temperature","type":"gauge","value":42.5}
}

// Example demonstrating path-based update and plain text read.
func Example_updatePathAndGetValue() {
	r := chi.NewRouter()
	storage := metrics.NewMemStorage()
	h := handlers.NewHandlers(storage)
	srv.SetMetricRoutes(r, h)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Update counter via path
	resp1, err := http.Post(ts.URL+"/update/counter/hits/10", "text/plain", nil)
	if err != nil {
		return
	}
	defer resp1.Body.Close()

	// Read back as plain text
	resp, err := http.Get(ts.URL + "/value/counter/hits")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	fmt.Println(string(bytes.TrimSpace(b)))
	// Output:
	// 10
}
