package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type MetricType string

const (
	Gauge   MetricType = "gauge"
	Counter MetricType = "counter"
)

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

type GaugeMetric struct {
	Value float64
}

type CounterMetric struct {
	Value int64
}

type MemStorage struct {
	gauges   map[string]GaugeMetric
	counters map[string]CounterMetric
	mu       sync.Mutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]GaugeMetric),
		counters: make(map[string]CounterMetric),
	}
}

func (ms *MemStorage) UpdateGauge(name string, value float64) {
	ms.gauges[name] = GaugeMetric{Value: value}
}

func (ms *MemStorage) UpdateCounter(name string, value int64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if existing, exists := ms.counters[name]; exists {
		value += existing.Value
	}
	ms.counters[name] = CounterMetric{Value: value}
}

func (ms *MemStorage) GetGauge(name string) (GaugeMetric, bool) {
	metric, exists := ms.gauges[name]
	return metric, exists
}

func (ms *MemStorage) GetCounter(name string) (CounterMetric, bool) {
	metric, exists := ms.counters[name]
	return metric, exists
}

type Server struct {
	storage *MemStorage
	tmpl    *template.Template
	logger  *zap.Logger // добавляем logger
}

func NewServer(storage *MemStorage, logger *zap.Logger) *Server {
	server := &Server{storage: storage, logger: logger}
	server.InitTemplate()
	return server
}

// Middleware для логирования запросов и ответов
func (s *Server) Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Подготовим кастомный ResponseWriter для получения информации о статусе и размере ответа
		ww := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		// Логируем информацию о запросе и ответе
		s.logger.Info("Handled request",
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.Duration("duration", duration),
			zap.Int("status", ww.status),
			zap.Int64("response_size", ww.size),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
	size   int64
}

func (ww *statusWriter) WriteHeader(status int) {
	ww.status = status
	ww.ResponseWriter.WriteHeader(status)
}

func (ww *statusWriter) Write(p []byte) (int, error) {
	size, err := ww.ResponseWriter.Write(p)
	ww.size += int64(size)
	return size, err
}

func (s *Server) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")
	metricValue := chi.URLParam(r, "value")

	if metricName == "" {
		http.Error(w, "Metric name not provided", http.StatusNotFound)
		return
	}

	switch MetricType(metricType) {
	case Gauge:
		value, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			http.Error(w, "Invalid metric value for gauge", http.StatusBadRequest)
			return
		}
		s.storage.UpdateGauge(metricName, value)
	case Counter:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			http.Error(w, "Invalid metric value for counter", http.StatusBadRequest)
			return
		}
		s.storage.UpdateCounter(metricName, value)
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) UpdateHandlerJson(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	var request Metrics
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.ID == "" {
		http.Error(w, "Metric ID is required", http.StatusBadRequest)
		return
	}

	// Создаём структуру для ответа
	response := Metrics{
		ID:    request.ID,
		MType: request.MType,
	}

	switch request.MType {
	case "gauge":
		// Обновляем значение метрики типа Gauge
		if request.Value == nil {
			http.Error(w, "Value is required for gauge", http.StatusBadRequest)
			return
		}
		s.storage.UpdateGauge(request.ID, *request.Value)
		response.Value = request.Value
	case "counter":
		// Обновляем значение метрики типа Counter
		if request.Delta == nil {
			http.Error(w, "Delta is required for counter", http.StatusBadRequest)
			return
		}
		s.storage.UpdateCounter(request.ID, *request.Delta)
		response.Delta = request.Delta
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	// Отправляем ответ с актуальными значениями
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func formatNumber(num float64) string {
	rounded := strconv.FormatFloat(num, 'f', 3, 64)
	rounded = strings.TrimRight(rounded, "0")
	rounded = strings.TrimRight(rounded, ".")
	return rounded
}

func (s *Server) GetValueHandler(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	metricName := chi.URLParam(r, "name")

	if metricName == "" {
		http.Error(w, "Metric name not provided", http.StatusNotFound)
		return
	}

	switch MetricType(metricType) {
	case Gauge:
		if metric, exists := s.storage.GetGauge(metricName); exists {
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, formatNumber(metric.Value))
		} else {
			http.Error(w, "Metric not found", http.StatusNotFound)
		}
	case Counter:
		if metric, exists := s.storage.GetCounter(metricName); exists {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(strconv.Itoa(int(metric.Value))))
		} else {
			http.Error(w, "Metric not found", http.StatusNotFound)
		}
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
	}
}

func (s *Server) GetValueHandlerPost(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Content-Type") != "application/json" {
		http.Error(w, "Invalid Content-Type", http.StatusBadRequest)
		return
	}

	var request Metrics
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.MType == "" {
		http.Error(w, "Metric name not provided", http.StatusNotFound)
		return
	}

	// Создаём структуру для ответа
	response := Metrics{
		ID:    request.ID,
		MType: request.MType,
	}

	// Проверка на существование метрики
	switch MetricType(request.MType) {
	case Gauge:
		if metric, exists := s.storage.GetGauge(request.ID); exists {
			response.Value = &metric.Value
		} else {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
	case Counter:
		if metric, exists := s.storage.GetCounter(request.ID); exists {
			response.Delta = &metric.Value
		} else {
			http.Error(w, "Metric not found", http.StatusNotFound)
			return
		}
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}

func (s *Server) InitTemplate() {
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
	s.tmpl = template.Must(template.New("metrics").Parse(tmpl))
}

func (s *Server) RootHandler(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Gauges   map[string]float64
		Counters map[string]int64
	}{
		Gauges:   make(map[string]float64),
		Counters: make(map[string]int64),
	}

	for name, metric := range s.storage.gauges {
		data.Gauges[name] = metric.Value
	}

	for name, metric := range s.storage.counters {
		data.Counters[name] = metric.Value
	}

	s.tmpl.Execute(w, data)
}

func main() {
	// Настройка логирования с использованием zap
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	addressEnv := os.Getenv("ADDRESS")

	defaultAddress := "localhost:8080"
	address := flag.String("a", defaultAddress, "HTTP server address (without http:// or https://)")
	flag.Parse()

	if addressEnv != "" {
		*address = addressEnv
	}

	storage := NewMemStorage()
	server := NewServer(storage, logger)

	r := chi.NewRouter()

	// Добавляем middleware для логирования
	r.Use(server.Logger)

	r.Post("/update/{type}/{name}/{value}", server.UpdateHandler)
	r.Post("/update/", server.UpdateHandlerJson)
	r.Post("/value/", server.GetValueHandlerPost)
	r.Get("/value/{type}/{name}", server.GetValueHandler)
	r.Get("/", server.RootHandler)

	logger.Info("Starting server", zap.String("address", *address))
	if err := http.ListenAndServe(*address, r); err != nil {
		logger.Error("Error starting server", zap.Error(err))
	}
}
