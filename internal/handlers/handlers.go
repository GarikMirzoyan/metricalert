package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/GarikMirzoyan/metricalert/internal/database"
	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/GarikMirzoyan/metricalert/internal/repositories"
	"github.com/go-chi/chi"
)

// Handlers содержит зависимости
type Handlers struct {
	ms   *metrics.MemStorage
	tmpl *template.Template
}

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

type DBHandler struct {
	DBConn database.DBConn
	tmpl   *template.Template
}

func NewDBHandler(DBConn database.DBConn) *DBHandler {
	DBHandler := &DBHandler{DBConn: DBConn, tmpl: InitTemplate()}

	return DBHandler
}

// NewHandlers создаёт новый Handlers
func NewHandlers(ms *metrics.MemStorage) *Handlers {
	handlers := &Handlers{ms: ms, tmpl: InitTemplate()}

	return handlers
}

func (h *Handlers) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	fmt.Println("Тип:", metricType)
	fmt.Println("Имя:", metricName)
	fmt.Println("Значение:", metricValue)

	if metricName == "" {
		http.Error(w, "Metric name not provided", http.StatusNotFound)
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

func (h *Handlers) UpdateHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	response, err := h.ms.UpdateMetricsFromJSON(r)
	if err != nil {
		switch err {
		case metrics.ErrInvalidMetricValue:
			http.Error(w, "Value is required for gauge", http.StatusBadRequest)
		case metrics.ErrInvalidMetricDelta:
			http.Error(w, "Value is required for delta", http.StatusBadRequest)
		case metrics.ErrInvalidMetricType:
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

func (h *Handlers) GetValueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	if metricName == "" {
		http.Error(w, "Metric name not provided", http.StatusNotFound)
		return
	}

	value, err := h.ms.GetMetricValue(metricType, metricName)
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

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(value))
}

func (h *Handlers) GetValueHandlerPost(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func (h *Handlers) RootHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Error rendering template", http.StatusInternalServerError)
	}
}

func InitTemplate() *template.Template {
	const tmpl = `
		<!DOCTYPE html>
		<html>
		<head>
			<title>Metrics</title>
		</head>
		<body>
			<h1>Metrics</h1>
			<ul>
			{{range $key, $value := .Gauges}}
				<li>{{$key}}: {{$value}}</li>
			{{end}}
			{{range $key, $value := .Counters}}
				<li>{{$key}}: {{$value}}</li>
			{{end}}
			</ul>
		</body>
		</html>
	`
	return template.Must(template.New("metrics").Parse(tmpl))
}

func (h *DBHandler) PingDBHandler(w http.ResponseWriter, r *http.Request) {
	err := h.DBConn.Ping(context.Background())

	if err != nil {
		http.Error(w, fmt.Sprintf("Произошла ошибка: %v", err), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (h *DBHandler) UpdateMetricDBHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *DBHandler) UpdateMetricDBHandlerJSON(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	metricRepository := repositories.NewMetricRepository(h.DBConn)

	response, err := metrics.UpdateMetricsDBFromJSON(r, metricRepository)
	if err != nil {
		switch err {
		case metrics.ErrInvalidMetricValue:
			http.Error(w, "Value is required for gauge", http.StatusBadRequest)
		case metrics.ErrInvalidMetricDelta:
			http.Error(w, "Value is required for delta", http.StatusBadRequest)
		case metrics.ErrInvalidMetricType:
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

func (h *DBHandler) GetMetricValueDBHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *DBHandler) GetValueDBHandlerPost(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	metricRepository := repositories.NewMetricRepository(h.DBConn)

	response, err := metrics.GetMetricsDBFromJSON(r, metricRepository)
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
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func (h *DBHandler) RootDBHandler(w http.ResponseWriter, r *http.Request) {
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
	}
}
