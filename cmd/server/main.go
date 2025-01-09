package main

import (
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type MetricType string

const (
	Gauge   MetricType = "gauge"
	Counter MetricType = "counter"
)

type GaugeMetric struct {
	Value float64
}

type CounterMetric struct {
	Value int64
}

type MemStorage struct {
	gauges   map[string]GaugeMetric
	counters map[string]CounterMetric
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
}

func NewServer(storage *MemStorage) *Server {
	return &Server{storage: storage}
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

func round(value float64, precision int) float64 {
	factor := math.Pow(10, float64(precision))
	return math.Round(value*factor) / factor
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
			roundedValue := round(metric.Value, 3)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("%.3f", roundedValue)))
		} else {
			http.Error(w, "Metric not found", http.StatusNotFound)
		}
	case Counter:
		if metric, exists := s.storage.GetCounter(metricName); exists {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(fmt.Sprintf("%d", metric.Value)))
		} else {
			http.Error(w, "Metric not found", http.StatusNotFound)
		}
	default:
		http.Error(w, "Invalid metric type", http.StatusBadRequest)
	}
}

func (s *Server) RootHandler(w http.ResponseWriter, r *http.Request) {
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

	t := template.Must(template.New("metrics").Parse(tmpl))
	t.Execute(w, data)
}

func main() {
	storage := NewMemStorage()
	server := NewServer(storage)

	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", server.UpdateHandler)
	r.Get("/value/{type}/{name}", server.GetValueHandler)
	r.Get("/", server.RootHandler)

	fmt.Println("Starting server at :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
