package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/GarikMirzoyan/metricalert/internal/database"
	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/GarikMirzoyan/metricalert/internal/repositories"
	"github.com/GarikMirzoyan/metricalert/internal/utils"
	"github.com/go-chi/chi"
)

type DBHandler struct {
	DBConn database.DBConn
	tmpl   *template.Template
}

func NewDBHandlers(DBConn database.DBConn) *DBHandler {
	DBHandler := &DBHandler{DBConn: DBConn, tmpl: utils.InitTemplate()}

	return DBHandler
}

func (h *DBHandler) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	metricRepository := repositories.NewMetricRepository(h.DBConn)

	err := metricRepository.Update(metricType, metricName, metricValue, r.Context())

	if err != nil {
		http.Error(w, fmt.Sprintf("Произошла ошибка: %v", err), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *DBHandler) UpdateHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	metricRepository := repositories.NewMetricRepository(h.DBConn)

	response, err := metrics.UpdateMetricsDBFromJSON(r, metricRepository)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func (h *DBHandler) GetValueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	if metricName == "" {
		http.Error(w, "Metric name not provided", http.StatusNotFound)
	}

	metricRepository := repositories.NewMetricRepository(h.DBConn)

	var (
		value any
		err   error
	)

	if metricType == "gauge" {
		value, err = metricRepository.GetGaugeValue(metricName, r.Context())
	} else {
		value, err = metricRepository.GetCounterValue(metricName, r.Context())
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Произошла ошибка: %v", err), http.StatusInternalServerError)
	}

	valStr := fmt.Sprintf("%v", value)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(valStr))
}

func (h *DBHandler) GetValueHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	metricRepository := repositories.NewMetricRepository(h.DBConn)

	response, err := metrics.GetMetricsDBFromJSON(r, metricRepository)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func (h *DBHandler) RootHandler(w http.ResponseWriter, r *http.Request) {
	metricRepository := repositories.NewMetricRepository(h.DBConn)

	gauges, counters, err := metricRepository.GetAllMetrics(r.Context())
	if err != nil {
		http.Error(w, "Ошибка при получении метрик из БД", http.StatusInternalServerError)
		return
	}

	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   gauges,
		Counters: counters,
	}

	w.Header().Set("Content-Type", "text/html")

	if err := h.tmpl.Execute(w, data); err != nil {
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
		return
	}
}

func (h *DBHandler) BatchMetricsUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	metricRepository := repositories.NewMetricRepository(h.DBConn)

	err := metrics.BatchMetricsUpdate(r, metricRepository)
	if err != nil {
		http.Error(w, fmt.Sprintf("Произошла ошибка: %v", err), http.StatusInternalServerError)
		return
	}

	// Отправляем ответ с актуальными значениями
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// func (h *DBHandler) PingDBHandler(w http.ResponseWriter, r *http.Request) {
// 	err := h.DBConn.Ping(context.Background())

// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Произошла ошибка: %v", err), http.StatusInternalServerError)
// 	}

// 	w.WriteHeader(http.StatusOK)
// }
