package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	dto "github.com/GarikMirzoyan/metricalert/internal/DTO"
	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/GarikMirzoyan/metricalert/internal/models"
	"github.com/GarikMirzoyan/metricalert/internal/utils"
	"github.com/go-chi/chi"
)

// Handler groups HTTP handlers for metrics API and keeps dependencies.
type Handler struct {
	ms   metrics.MetricStorage
	tmpl *template.Template
}

// NewHandlers constructs metric handlers with provided storage.
func NewHandlers(ms metrics.MetricStorage) *Handler {
	DBHandler := &Handler{ms: ms, tmpl: utils.InitTemplate()}

	return DBHandler
}

// UpdateHandler handles path-based metric updates:
// POST /update/{type}/{name}/{value}
func (h *Handler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	metric, err := metrics.NewMetric(metricType, metricName, metricValue)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = h.ms.Update(metric, r.Context())

	if err != nil {
		switch err {
		case metrics.ErrInvalidMetricType:
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
		case metrics.ErrInvalidMetricValue:
			http.Error(w, "Invalid metric value", http.StatusBadRequest)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// UpdateHandlerJSON handles JSON metric updates via POST /update/.
func (h *Handler) UpdateHandlerJSON(w http.ResponseWriter, r *http.Request) {
	// Проверка Content-Type
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	var request dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
	}

	if request.ID == "" {
		http.Error(w, "metric ID is required", http.StatusBadRequest)
	}

	metric, err := metrics.NewMetricFromDTO(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := h.ms.UpdateJSON(metric, r.Context())
	if err != nil {
		switch {
		case errors.Is(err, metrics.ErrInvalidMetricValue):
			http.Error(w, "Value is required for gauge", http.StatusBadRequest)
		case errors.Is(err, metrics.ErrInvalidMetricDelta):
			http.Error(w, "Value is required for delta", http.StatusBadRequest)
		case errors.Is(err, metrics.ErrInvalidMetricType):
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Устанавливаем правильный Content-Type для JSON ответа
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Кодируем и отправляем ответ
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Ошибка получения данных", http.StatusInternalServerError)
	}
}

// GetValueHandler returns metric value as plain text by type and name.
func (h *Handler) GetValueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	if metricName == "" {
		http.Error(w, "Имя метрики не передано", http.StatusBadRequest)
		return
	}

	value, err := h.ms.GetValue(metricType, metricName, r.Context())
	if err != nil {
		switch {
		case errors.Is(err, metrics.ErrMetricNotFound):
			http.Error(w, "Metric not found", http.StatusNotFound)
		case errors.Is(err, metrics.ErrInvalidMetricType):
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(value))
}

// GetValueHandlerJSON returns metric value in JSON form for a given DTO.
func (h *Handler) GetValueHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	var request dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
	}

	if request.ID == "" {
		http.Error(w, "invalid metric type", http.StatusBadRequest)
	}

	metric, err := metrics.NewMetricFromDTO(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	response, err := h.ms.GetJSON(metric, r.Context())
	if err != nil {
		switch {
		case errors.Is(err, metrics.ErrMetricNotFound):
			http.Error(w, "Metric not found", http.StatusNotFound)
		case errors.Is(err, metrics.ErrInvalidMetricType):
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
		default:
			http.Error(w, fmt.Sprintf("Internal server error: %v", err), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Ошибка получения данных", http.StatusInternalServerError)
	}
}

// RootHandler renders an HTML page with all gauges and counters.
func (h *Handler) RootHandler(w http.ResponseWriter, r *http.Request) {
	gauges, counters, err := h.ms.GetAll(r.Context())
	if err != nil {
		http.Error(w, "Ошибка при получении метрик", http.StatusInternalServerError)
		return
	}

	data := struct {
		Gauges   map[string]models.GaugeMetric
		Counters map[string]models.CounterMetric
	}{
		Gauges:   gauges,
		Counters: counters,
	}

	w.Header().Set("Content-Type", "text/html")

	if err := h.tmpl.Execute(w, data); err != nil {
		http.Error(w, "Ошибка при инициализации шаблона", http.StatusInternalServerError)
	}
}

// BatchMetricsUpdateHandler updates multiple metrics in a single request.
func (h *Handler) BatchMetricsUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	var metricsDTO []dto.Metrics
	if err := json.NewDecoder(r.Body).Decode(&metricsDTO); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Преобразуем DTO в map[string]models.Metric
	var metricsList []models.Metric
	for _, dto := range metricsDTO {
		metric, err := metrics.NewMetricFromDTO(dto)
		if err != nil {
			http.Error(w, fmt.Sprintf("Invalid metric: %v", err), http.StatusBadRequest)
			return
		}
		metricsList = append(metricsList, metric)
	}

	// Обновляем метрики
	response, err := h.ms.UpdateBatchJSON(metricsList, r.Context())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
