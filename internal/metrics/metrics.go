package metrics

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"runtime"
	"strconv"

	// Используем псевдонимы для избежания конфликта имен
	"github.com/GarikMirzoyan/metricalert/internal/DTO"
	agentConfig "github.com/GarikMirzoyan/metricalert/internal/agent/config"
	"github.com/GarikMirzoyan/metricalert/internal/constants"
	"github.com/GarikMirzoyan/metricalert/internal/models"
	"github.com/GarikMirzoyan/metricalert/internal/retry"
)

type Gauge float64
type Counter int64

var (
	ErrMetricNotFound     = errors.New("metric not found")
	ErrInvalidMetricType  = errors.New("invalid metric type")
	ErrInvalidMetricValue = errors.New("invalid metric value")
	ErrInvalidMetricDelta = errors.New("invalid metric delta")
	ErrInvalidJSON        = errors.New("invalid JSON")
	ErrInvalidMetricID    = errors.New("metric ID is required")
)

func NewMetric(metricType, metricName, metricValue string) (models.Metric, error) {
	switch constants.MetricType(metricType) {
	case constants.GaugeName:
		val, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid gauge value: %w", err)
		}
		return &models.GaugeMetric{Name: metricName, Type: constants.GaugeName, Value: val}, nil

	case constants.CounterName:
		val, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid counter value: %w", err)
		}
		return &models.CounterMetric{Name: metricName, Type: constants.GaugeName, Value: val}, nil

	default:
		return nil, fmt.Errorf("unknown metric type: %s", metricType)
	}
}

func NewMetricFromDTO(metricDTO DTO.Metrics) (models.Metric, error) {
	switch constants.MetricType(metricDTO.MType) {
	case constants.GaugeName:
		return &models.GaugeMetric{
			Name: metricDTO.ID,
			Type: constants.GaugeName,
			Value: func() float64 {
				if metricDTO.Value != nil {
					return *metricDTO.Value
				}
				return 0
			}(),
		}, nil

	case constants.CounterName:
		return &models.CounterMetric{
			Name: metricDTO.ID,
			Type: constants.CounterName,
			Value: func() int64 {
				if metricDTO.Delta != nil {
					return *metricDTO.Delta
				}
				return 0
			}(),
		}, nil

	default:
		return nil, fmt.Errorf("unknown metric type: %s", metricDTO.MType)
	}
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

func SendMetric(metric DTO.Metrics, config agentConfig.Config) {
	url := fmt.Sprintf("%s/update/", config.Address)

	body, err := json.Marshal(metric)
	if err != nil {
		log.Printf("ошибка маршалинга JSON: %v", err)
		return
	}

	compressedBody, err := compressGzip(body)
	if err != nil {
		log.Printf("ошибка сжатия данных: %v", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedBody))
	if err != nil {
		log.Printf("ошибка создания HTTP-запроса: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("ошибка при отправке запроса: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("неуспешный статус ответа: %s", resp.Status)
	}
}

func SendBatchMetrics(metrics []DTO.Metrics, config agentConfig.Config) {
	url := fmt.Sprintf("%s/updates/", config.Address)

	body, err := json.Marshal(metrics)
	if err != nil {
		log.Printf("ошибка маршалинга JSON при отправке батча метрик: %v", err)
		return
	}

	compressedBody, err := compressGzip(body)
	if err != nil {
		log.Printf("ошибка сжатия тела запроса: %v", err)
		return
	}

	err = retry.WithBackoff(func() error {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedBody))
		if err != nil {
			return err // ошибка создания запроса — не retriable
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err // будет обработан как retriable, если это сетевой сбой
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			// временные ошибки сервера (например, 500, 503)
			return fmt.Errorf("server error: %s", resp.Status)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// другие ошибки (например, 400) — не повторяем
			return fmt.Errorf("non-retriable status: %s", resp.Status)
		}

		return nil // успех
	})

	if err != nil {
		log.Printf("не удалось отправить батч метрик после повторов: %v", err)
	}
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
