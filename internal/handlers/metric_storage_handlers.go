package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/GarikMirzoyan/metricalert/internal/utils"
	"github.com/go-chi/chi"
)

// Handlers содержит зависимости
type MemHandler struct {
	ms   *metrics.MemStorage
	tmpl *template.Template
}

func NewMemHandlers(ms *metrics.MemStorage) *MemHandler {
	DBHandler := &MemHandler{ms: ms, tmpl: utils.InitTemplate()}

	return DBHandler
}

func (h *MemHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	fmt.Println("Тип:", metricType)
	fmt.Println("Имя:", metricName)
	fmt.Println("Значение:", metricValue)

	if metricName == "" {
		http.Error(w, "Имя метрики не передано", http.StatusNotFound)
		return
	}

	err := h.ms.UpdateMetrics(metricType, metricName, metricValue)
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

func (h *MemHandler) UpdateHandlerJSON(w http.ResponseWriter, r *http.Request) {
	// Проверка Content-Type
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	response, err := h.ms.UpdateMetricsFromJSON(r)
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

func (h *MemHandler) GetValueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	if metricName == "" {
		http.Error(w, "Имя метрики не передано", http.StatusBadRequest)
		return
	}

	value, err := h.ms.GetMetricValue(metricType, metricName)
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

func (h *MemHandler) GetValueHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	response, err := h.ms.GetMetricsFromJSON(r)
	if err != nil {
		switch err {
		case metrics.ErrMetricNotFound:
			http.Error(w, "Metric not found", http.StatusNotFound)
		case metrics.ErrInvalidMetricType:
			http.Error(w, "Invalid metric type", http.StatusBadRequest)
		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Ошибка получения данных", http.StatusInternalServerError)
	}
}

func (h *MemHandler) RootHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем все метрики из хранилища
	gauges, counters := h.ms.GetAllMetrics()

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   gauges,
		Counters: counters,
	}

	w.Header().Set("Content-Type", "text/html")

	if err := h.tmpl.Execute(w, data); err != nil {
		http.Error(w, "Ошибка при инициализации шаблона", http.StatusInternalServerError)
	}
}

func (h *MemHandler) BatchMetricsUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	response, err := h.ms.UpdateBathMetricsFromJSON(r)
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

	// Отправляем ответ с актуальными значениями
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
