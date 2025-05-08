package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/go-chi/chi"
)

// Handlers содержит зависимости
type Handlers struct {
	ms   *metrics.MemStorage
	tmpl *template.Template
}

// NewHandlers создаёт новый Handlers
func NewHandlers(ms *metrics.MemStorage) *Handlers {
	handlers := &Handlers{ms: ms}

	handlers.InitTemplate()
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

	response, err := h.ms.UpdateMetricsFromJson(r)
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

	response, err := h.ms.GetMetricsFromJson(r)
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

func (h *Handlers) InitTemplate() {
	tmpl := `
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
	h.tmpl = template.Must(template.New("metrics").Parse(tmpl))
}
