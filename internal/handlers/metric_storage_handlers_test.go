package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/GarikMirzoyan/metricalert/internal/models"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
)

func TestUpdateHandler(t *testing.T) {
	ms := metrics.NewMemStorage()
	h := NewMemHandlers(ms)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateHandler)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/test_gauge/10.5", nil)
	rr := httptest.NewRecorder()

	fmt.Println("Запрос:", req.Method, req.URL.Path)

	r.ServeHTTP(rr, req)

	fmt.Println("Код ответа:", rr.Code)
	fmt.Println("Тело ответа:", rr.Body.String())

	// Проверка результата
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "OK")
}

func TestUpdateHandler_InvalidType(t *testing.T) {
	ms := metrics.NewMemStorage()
	h := NewMemHandlers(ms)

	r := chi.NewRouter()
	r.Post("/update/{type}/{name}/{value}", h.UpdateHandler)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge_invalid/test_gauge/10.5", nil)
	rr := httptest.NewRecorder()

	fmt.Println("Запрос:", req.Method, req.URL.Path)

	r.ServeHTTP(rr, req)

	fmt.Println("Код ответа:", rr.Code)
	fmt.Println("Тело ответа:", rr.Body.String())

	// Проверка результата
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Invalid metric type")
}

func TestUpdateHandlerJSON(t *testing.T) {
	ms := metrics.NewMemStorage()
	h := NewMemHandlers(ms)

	r := chi.NewRouter()
	r.Post("/update/", h.UpdateHandlerJSON)

	// Мокируем запрос с JSON телом
	body := `{"id":"test_gauge","type":"gauge","value":15.5}`
	req := httptest.NewRequest(http.MethodPost, "/update/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Проверка результата
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp models.Metrics
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "test_gauge", resp.ID)
	assert.Equal(t, "gauge", resp.MType)
	assert.Equal(t, 15.5, *resp.Value)
}

func TestGetValueHandler(t *testing.T) {
	ms := metrics.NewMemStorage()
	ms.UpdateMetrics("gauge", "test_gauge", "10.5")
	h := NewMemHandlers(ms)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", h.GetValueHandler)

	// Мокируем запрос
	req := httptest.NewRequest(http.MethodGet, "/value/gauge/test_gauge", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Проверка результата
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "10.5", rr.Body.String())
}

func TestGetValueHandler_InvalidType(t *testing.T) {
	ms := metrics.NewMemStorage()
	ms.UpdateMetrics("gauge", "test_gauge", "10.5")
	h := NewMemHandlers(ms)

	r := chi.NewRouter()
	r.Get("/value/{type}/{name}", h.GetValueHandler)

	// Мокируем запрос
	req := httptest.NewRequest(http.MethodGet, "/value/gauge_invalid/test_gauge", nil)
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Проверка результата
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestRootHandler(t *testing.T) {
	ms := metrics.NewMemStorage()
	ms.UpdateMetrics("gauge", "test_gauge", "10.5")
	ms.UpdateMetrics("counter", "test_counter", "5")
	h := NewMemHandlers(ms)

	// Мокируем запрос
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	// Вызов обработчика
	h.RootHandler(rr, req)

	// Проверка результата
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "test_gauge")
	assert.Contains(t, rr.Body.String(), "test_counter")
}

func TestGetValueHandlerPost(t *testing.T) {
	ms := metrics.NewMemStorage()
	ms.UpdateMetrics("gauge", "test_gauge", "10.5")
	h := NewMemHandlers(ms)

	// Мокируем запрос с JSON телом
	body := `{"id":"test_gauge","type":"gauge"}`

	r := chi.NewRouter()
	r.Post("/value/", h.GetValueHandlerJSON)

	req := httptest.NewRequest(http.MethodPost, "/value/", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	// Проверка результата
	assert.Equal(t, http.StatusOK, rr.Code)

	var resp models.Metrics
	err := json.NewDecoder(rr.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "test_gauge", resp.ID)
	assert.Equal(t, "gauge", resp.MType)
}
