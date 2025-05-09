package metrics

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	// Используем псевдонимы для избежания конфликта имен
	agentConfig "github.com/GarikMirzoyan/metricalert/internal/agent/config"
	"github.com/GarikMirzoyan/metricalert/internal/models"
	"github.com/GarikMirzoyan/metricalert/internal/repositories"
	serverConfig "github.com/GarikMirzoyan/metricalert/internal/server/config"

	"go.uber.org/zap"
)

type MetricType string

type Gauge float64
type Counter int64

const (
	GaugeName   MetricType = "gauge"
	CounterName MetricType = "counter"
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
	mu       sync.Mutex
}

var (
	ErrMetricNotFound     = errors.New("metric not found")
	ErrInvalidMetricType  = errors.New("invalid metric type")
	ErrInvalidMetricValue = errors.New("invalid metric value")
	ErrInvalidMetricDelta = errors.New("invalid metric delta")
	ErrInvalidJSON        = errors.New("invalid JSON")
	ErrInvalidMetricID    = errors.New("metric ID is required")
)

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

// Получаем метрики из структуры, хранящ статистику по памяти Go-приложения
func CollectMetrics() map[string]Gauge {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := map[string]Gauge{
		"Alloc":         Gauge(memStats.Alloc),
		"BuckHashSys":   Gauge(memStats.BuckHashSys),
		"Frees":         Gauge(memStats.Frees),
		"GCCPUFraction": Gauge(memStats.GCCPUFraction),
		"GCSys":         Gauge(memStats.GCSys),
		"HeapAlloc":     Gauge(memStats.HeapAlloc),
		"HeapIdle":      Gauge(memStats.HeapIdle),
		"HeapInuse":     Gauge(memStats.HeapInuse),
		"HeapObjects":   Gauge(memStats.HeapObjects),
		"HeapReleased":  Gauge(memStats.HeapReleased),
		"HeapSys":       Gauge(memStats.HeapSys),
		"LastGC":        Gauge(memStats.LastGC),
		"Mallocs":       Gauge(memStats.Mallocs),
		"NextGC":        Gauge(memStats.NextGC),
		"PauseTotalNs":  Gauge(memStats.PauseTotalNs),
		"StackInuse":    Gauge(memStats.StackInuse),
		"StackSys":      Gauge(memStats.StackSys),
		"Sys":           Gauge(memStats.Sys),
		"TotalAlloc":    Gauge(memStats.TotalAlloc),
		"RandomValue":   Gauge(rand.Float64()),
		"Lookups":       Gauge(memStats.Lookups),
		"MCacheInuse":   Gauge(memStats.MCacheInuse),
		"MCacheSys":     Gauge(memStats.MCacheSys),
		"MSpanInuse":    Gauge(memStats.MSpanInuse),
		"MSpanSys":      Gauge(memStats.MSpanSys),
		"NumForcedGC":   Gauge(memStats.NumForcedGC),
		"NumGC":         Gauge(memStats.NumGC),
		"OtherSys":      Gauge(memStats.OtherSys),
	}

	return metrics
}

// Загрузка метрик из файла
func (ms *MemStorage) LoadMetricsFromFile(config serverConfig.Config) error {
	if !config.Restore {
		return nil
	}

	file, err := os.Open(config.FileStoragePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	for {
		var metric models.Metrics
		if err := decoder.Decode(&metric); err != nil {
			if err.Error() == "EOF" {
				break
			}
			return err
		}

		switch MetricType(metric.MType) {
		case GaugeName:
			if metric.Value != nil {
				ms.UpdateGauge(metric.ID, *metric.Value)
			}
		case CounterName:
			if metric.Delta != nil {
				ms.UpdateCounter(metric.ID, *metric.Delta)
			}
		}
	}
	return nil
}

func (ms *MemStorage) SaveMetricsToFile(config serverConfig.Config) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	file, err := os.Create(config.FileStoragePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for name, gauge := range ms.gauges {
		metric := models.Metrics{
			ID:    name,
			MType: string(GaugeName),
			Value: &gauge.Value,
		}
		if err := encoder.Encode(metric); err != nil {
			return err
		}
	}

	for name, counter := range ms.counters {
		metric := models.Metrics{
			ID:    name,
			MType: string(CounterName),
			Delta: &counter.Value,
		}
		if err := encoder.Encode(metric); err != nil {
			return err
		}
	}

	return nil
}

// Функция для периодического сохранения метрик
func (ms *MemStorage) StartMetricSaving(config serverConfig.Config, logger *zap.Logger) {
	if config.StoreInterval == 0 {
		// Сохраняем синхронно
		if err := ms.SaveMetricsToFile(config); err != nil {
			logger.Error("Error saving metrics", zap.Error(err))
		}
	} else {
		ticker := time.NewTicker(config.StoreInterval)
		for range ticker.C {
			if err := ms.SaveMetricsToFile(config); err != nil {
				logger.Error("Error saving metrics", zap.Error(err))
			}
		}
	}
}

func (ms *MemStorage) UpdateMetrics(metricType, metricName, metricValue string) error {
	switch MetricType(metricType) {
	case GaugeName:
		value, err := strconv.ParseFloat(metricValue, 64)
		fmt.Println("Ошибка:", err)
		if err != nil {
			return ErrInvalidMetricValue
		}
		ms.UpdateGauge(metricName, value)
	case CounterName:
		value, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return ErrInvalidMetricValue
		}
		ms.UpdateCounter(metricName, value)
	default:
		return ErrInvalidMetricType
	}
	return nil
}

func (ms *MemStorage) UpdateMetricsFromJSON(r *http.Request) (models.Metrics, error) {

	var request models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return models.Metrics{}, ErrInvalidJSON
	}

	if request.ID == "" {
		return models.Metrics{}, ErrInvalidMetricID
	}

	// Создаём структуру для ответа
	response := models.Metrics{
		ID:    request.ID,
		MType: request.MType,
	}

	switch MetricType(request.MType) {
	case GaugeName:
		// Обновляем значение метрики типа Gauge
		if request.Value == nil {
			return models.Metrics{}, ErrInvalidMetricDelta
		}
		ms.UpdateGauge(request.ID, *request.Value)
		response.Value = request.Value
	case CounterName:
		// Обновляем значение метрики типа Counter
		if request.Delta == nil {
			return models.Metrics{}, ErrInvalidMetricDelta
		}
		ms.UpdateCounter(request.ID, *request.Delta)
		response.Delta = request.Delta
	default:
		return models.Metrics{}, ErrInvalidMetricType
	}

	return response, nil
}

func (ms *MemStorage) GetMetricsFromJSON(r *http.Request) (models.Metrics, error) {

	var request models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return models.Metrics{}, ErrInvalidJSON
	}
	if request.MType == "" {
		return models.Metrics{}, ErrInvalidMetricType
	}

	// Создаём структуру для ответа
	response := models.Metrics{
		ID:    request.ID,
		MType: request.MType,
	}

	// Проверка на существование метрики
	switch MetricType(request.MType) {
	case GaugeName:
		if metric, exists := ms.GetGauge(request.ID); exists {
			response.Value = &metric.Value
		} else {
			return models.Metrics{}, ErrMetricNotFound
		}
	case CounterName:
		if metric, exists := ms.GetCounter(request.ID); exists {
			response.Delta = &metric.Value
		} else {
			return models.Metrics{}, ErrMetricNotFound
		}
	default:
		return models.Metrics{}, ErrInvalidMetricType
	}

	return response, nil
}

// GetMetricValue получает значение метрики и возвращает его как строку
func (ms *MemStorage) GetMetricValue(metricType, metricName string) (string, error) {
	switch MetricType(metricType) {
	case GaugeName:
		if metric, exists := ms.GetGauge(metricName); exists {
			return formatNumber(metric.Value), nil
		}
	case CounterName:
		if metric, exists := ms.GetCounter(metricName); exists {
			return strconv.Itoa(int(metric.Value)), nil
		}
	default:
		return "", ErrInvalidMetricType
	}
	return "", ErrMetricNotFound
}

func formatNumber(num float64) string {
	rounded := strconv.FormatFloat(num, 'f', 3, 64)
	rounded = strings.TrimRight(rounded, "0")
	rounded = strings.TrimRight(rounded, ".")
	return rounded
}

func (ms *MemStorage) GetAllMetrics() (map[string]float64, map[string]int64) {
	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	// Копируем данные из хранилища
	for name, metric := range ms.gauges {
		gauges[name] = metric.Value
	}

	for name, metric := range ms.counters {
		counters[name] = metric.Value
	}

	return gauges, counters
}

func SendMetric(metric models.Metrics, config agentConfig.Config) {
	url := fmt.Sprintf("%s/update/", config.Address)

	body, err := json.Marshal(metric)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		return
	}

	// Сжимаем данные перед отправкой
	compressedBody, err := compressGzip(body)
	if err != nil {
		fmt.Printf("Error compressing data: %v\n", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedBody))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Устанавливаем заголовки для gzip
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()
}

func SendBatchMetrics(metrics []models.Metrics, config agentConfig.Config) {
	url := fmt.Sprintf("%s/updates/", config.Address)

	body, err := json.Marshal(metrics)

	// Сжимаем данные перед отправкой
	compressedBody, err := compressGzip(body)
	if err != nil {
		fmt.Printf("Error compressing data: %v\n", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedBody))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Устанавливаем заголовки для gzip
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()
}

// Функция для сжатия данных в формате gzip
func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, err
	}
	gzipWriter.Close()
	return buf.Bytes(), nil
}

func UpdateMetricsDBFromJSON(r *http.Request, mr *repositories.MetricRepository) (models.Metrics, error) {

	var request models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return models.Metrics{}, ErrInvalidJSON
	}

	if request.ID == "" {
		return models.Metrics{}, ErrInvalidMetricID
	}

	// Создаём структуру для ответа
	response := models.Metrics{
		ID:    request.ID,
		MType: request.MType,
	}

	switch MetricType(request.MType) {
	case GaugeName:
		// Обновляем значение метрики типа Gauge
		if request.Value == nil {
			return models.Metrics{}, ErrInvalidMetricDelta
		}
		err := mr.Update("gauge", request.ID, fmt.Sprintf("%f", *request.Value), r.Context())
		if err != nil {
			return models.Metrics{}, err
		}
		response.Value = request.Value
	case CounterName:
		// Обновляем значение метрики типа Counter
		if request.Delta == nil {
			return models.Metrics{}, ErrInvalidMetricDelta
		}
		err := mr.Update("counter", request.ID, fmt.Sprintf("%d", *request.Delta), r.Context())
		if err != nil {
			return models.Metrics{}, err
		}
		response.Delta = request.Delta
	default:
		return models.Metrics{}, ErrInvalidMetricType
	}

	return response, nil
}

func GetMetricsDBFromJSON(r *http.Request, mr *repositories.MetricRepository) (models.Metrics, error) {
	var request models.Metrics
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return models.Metrics{}, ErrInvalidJSON
	}
	if request.MType == "" {
		return models.Metrics{}, ErrInvalidMetricType
	}

	// Создаём структуру для ответа
	response := models.Metrics{
		ID:    request.ID,
		MType: request.MType,
	}

	// Проверка на существование метрики
	switch MetricType(request.MType) {
	case GaugeName:
		val, err := mr.GetGaugeValue(request.ID, r.Context())
		if err != nil {
			return models.Metrics{}, ErrMetricNotFound
		}
		response.Value = &val // val уже float64

	case CounterName:
		val, err := mr.GetCounterValue(request.ID, r.Context()) // int64
		if err != nil {
			return models.Metrics{}, ErrMetricNotFound
		}
		response.Delta = &val

	default:
		return models.Metrics{}, ErrInvalidMetricType
	}

	return response, nil
}

func BatchMetricsUpdate(r *http.Request, mr *repositories.MetricRepository) error {

	if r.Body == nil {
		return errors.New("request body is empty")
	}
	defer r.Body.Close()

	var metrics []models.Metrics

	if err := json.NewDecoder(r.Body).Decode(&metrics); err != nil {
		return fmt.Errorf("failed to decode metrics: %w", err)
	}

	if len(metrics) == 0 {
		return nil // Нет метрик — ничего не делаем
	}

	if err := mr.BatchUpdate(metrics, r.Context()); err != nil {
		return fmt.Errorf("failed to batch update metrics: %w", err)
	}

	return nil
}
