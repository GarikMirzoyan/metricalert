package metrics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	dto "github.com/GarikMirzoyan/metricalert/internal/DTO"
	"github.com/GarikMirzoyan/metricalert/internal/constants"
	"github.com/GarikMirzoyan/metricalert/internal/models"
	serverConfig "github.com/GarikMirzoyan/metricalert/internal/server/config"
	"github.com/GarikMirzoyan/metricalert/internal/utils"
	"go.uber.org/zap"
)

type MemStorage struct {
	gauges   map[string]models.GaugeMetric
	counters map[string]models.CounterMetric
	mu       sync.Mutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]models.GaugeMetric),
		counters: make(map[string]models.CounterMetric),
	}
}

func (ms *MemStorage) Update(metric models.Metric, ctx context.Context) error {
	switch m := metric.(type) {
	case *models.GaugeMetric:
		return ms.UpdateGauge(m, ctx)
	case *models.CounterMetric:
		return ms.UpdateCounter(m, ctx)
	default:
		return fmt.Errorf("invalid metric type")
	}
}

func (ms *MemStorage) UpdateJSON(metric models.Metric, ctx context.Context) (dto.Metrics, error) {
	// Проверка на nil значение
	if metric.GetValue() == nil {
		return dto.Metrics{}, ErrInvalidMetricDelta
	}

	// Создание ответа
	response := dto.Metrics{
		ID:    metric.GetName(),
		MType: string(metric.GetType()),
	}

	switch m := metric.(type) {
	case *models.GaugeMetric:
		if err := ms.UpdateGauge(m, ctx); err != nil {
			return dto.Metrics{}, fmt.Errorf("failed to update gauge: %w", err)
		}
		value, ok := metric.GetValue().(float64)
		if !ok {
			return dto.Metrics{}, fmt.Errorf("invalid type assertion: expected *float64")
		}
		response.Value = &value

	case *models.CounterMetric:
		if err := ms.UpdateCounter(m, ctx); err != nil {
			return dto.Metrics{}, fmt.Errorf("failed to update counter: %w", err)
		}
		delta, ok := metric.GetValue().(int64)
		if !ok {
			return dto.Metrics{}, fmt.Errorf("invalid type assertion: expected *int64")
		}
		response.Delta = &delta

	default:
		return dto.Metrics{}, ErrInvalidMetricType
	}

	return response, nil
}

func (ms *MemStorage) GetValue(metricType, metricName string, ctx context.Context) (string, error) {
	switch constants.MetricType(metricType) {
	case constants.GaugeName:
		metric, err := ms.GetGauge(metricName, ctx)
		if err != nil {
			return "", err
		}
		return utils.FormatNumber(metric.Value), nil

	case constants.CounterName:
		metric, err := ms.GetCounter(metricName, ctx)
		if err != nil {
			return "", err
		}
		return strconv.FormatInt(metric.Value, 10), nil

	default:
		return "", ErrInvalidMetricType
	}
}

func (ms *MemStorage) GetJSON(metric models.Metric, ctx context.Context) (dto.Metrics, error) {
	// Создание ответа
	response := dto.Metrics{
		ID:    metric.GetName(),
		MType: string(metric.GetType()),
	}

	switch m := metric.(type) {
	case *models.GaugeMetric:
		metric, err := ms.GetGauge(m.GetName(), ctx)
		if err != nil {
			return dto.Metrics{}, err
		}

		value, ok := metric.GetValue().(float64)

		if !ok {
			return dto.Metrics{}, fmt.Errorf("invalid type assertion: expected *float64")
		}

		response.Value = &value

		return response, nil

	case *models.CounterMetric:
		metric, err := ms.GetCounter(m.GetName(), ctx)

		if err != nil {
			return dto.Metrics{}, err
		}

		delta, ok := metric.GetValue().(int64)

		if !ok {
			return dto.Metrics{}, fmt.Errorf("invalid type assertion: expected *int64")
		}
		response.Delta = &delta

		return response, nil
	default:
		return dto.Metrics{}, ErrInvalidMetricType
	}
}

func (ms *MemStorage) UpdateGauge(metric *models.GaugeMetric, ctx context.Context) error {
	if metric.Name == "" {
		return fmt.Errorf("metric name is empty")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.gauges[metric.Name] = *metric
	return nil
}

func (ms *MemStorage) UpdateCounter(metric *models.CounterMetric, ctx context.Context) error {
	if metric.Name == "" {
		return fmt.Errorf("metric name is empty")
	}

	ms.mu.Lock()
	defer ms.mu.Unlock()

	if existing, exists := ms.counters[metric.Name]; exists {
		metric.Value += existing.Value
	}
	ms.counters[metric.Name] = *metric
	return nil
}

func (ms *MemStorage) GetGauge(name string, ctx context.Context) (models.GaugeMetric, error) {
	metric, exists := ms.gauges[name]
	if !exists {
		return models.GaugeMetric{}, ErrMetricNotFound
	}
	return metric, nil
}

func (ms *MemStorage) GetCounter(name string, ctx context.Context) (models.CounterMetric, error) {
	metric, exists := ms.counters[name]
	if !exists {
		return models.CounterMetric{}, ErrMetricNotFound
	}
	return metric, nil
}

func (ms *MemStorage) GetAll(ctx context.Context) (map[string]models.GaugeMetric, map[string]models.CounterMetric, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	gauges := make(map[string]models.GaugeMetric)
	counters := make(map[string]models.CounterMetric)

	for name, metric := range ms.gauges {
		gauges[name] = metric
	}

	for name, metric := range ms.counters {
		counters[name] = metric
	}

	return gauges, counters, nil
}

func (ms *MemStorage) UpdateBatchJSON(metrics map[string]models.Metric, ctx context.Context) (map[string]dto.Metrics, error) {
	responses := make(map[string]dto.Metrics)

	for key, metric := range metrics {
		if key == "" {
			return nil, ErrInvalidMetricID
		}

		response := dto.Metrics{
			ID:    key,
			MType: string(metric.GetType()),
		}

		switch m := metric.(type) {
		case *models.GaugeMetric:
			ms.UpdateGauge(m, ctx)
			response.Value = &m.Value
		case *models.CounterMetric:
			ms.UpdateCounter(m, ctx)
			response.Delta = &m.Value
		default:
			return nil, ErrInvalidMetricType
		}

		responses[key] = response
	}

	return responses, nil
}

func (ms *MemStorage) LoadMetricsFromFile(config serverConfig.Config) error {
	if !config.Restore {
		return nil
	}

	file, err := os.Open(config.FileStoragePath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл для чтения метрик: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	ctx := context.Background()

	for {
		var metricDTO dto.Metrics
		if err := decoder.Decode(&metricDTO); err != nil {
			if errors.Is(err, io.EOF) {
				break // всё успешно прочитано
			}
			return fmt.Errorf("ошибка при декодировании JSON: %w", err)
		}

		switch constants.MetricType(metricDTO.MType) {
		case constants.GaugeName:
			if metricDTO.Value != nil {
				metric := &models.GaugeMetric{
					Name:  metricDTO.ID,
					Type:  constants.GaugeName,
					Value: *metricDTO.Value,
				}
				if err := ms.UpdateGauge(metric, ctx); err != nil {
					return fmt.Errorf("ошибка при обновлении gauge метрики: %w", err)
				}
			}

		case constants.CounterName:
			if metricDTO.Delta != nil {
				metric := &models.CounterMetric{
					Name:  metricDTO.ID,
					Type:  constants.CounterName,
					Value: *metricDTO.Delta,
				}
				if err := ms.UpdateCounter(metric, ctx); err != nil {
					return fmt.Errorf("ошибка при обновлении counter метрики: %w", err)
				}
			}

		default:
			return fmt.Errorf("неизвестный тип метрики: %s", metricDTO.MType)
		}
	}

	return nil
}

func (ms *MemStorage) SaveMetricsToFile(config serverConfig.Config) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Создание файла
	file, err := os.Create(config.FileStoragePath)
	if err != nil {
		return fmt.Errorf("не удалось создать файл для записи метрик: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	// Сохраняем метрики Gauge
	for name, gauge := range ms.gauges {
		metric := dto.Metrics{
			ID:    name,
			MType: string(constants.GaugeName),
			Value: &gauge.Value,
		}
		if err := encoder.Encode(metric); err != nil {
			return fmt.Errorf("ошибка при записи метрики %s в файл: %w", name, err)
		}
	}

	// Сохраняем метрики Counter
	for name, counter := range ms.counters {
		metric := dto.Metrics{
			ID:    name,
			MType: string(constants.CounterName),
			Delta: &counter.Value,
		}
		if err := encoder.Encode(metric); err != nil {
			return fmt.Errorf("ошибка при записи метрики %s в файл: %w", name, err)
		}
	}

	return nil
}

// Функция для периодического сохранения метрик
func (ms *MemStorage) StartMetricSaving(config serverConfig.Config, logger *zap.Logger) {
	if config.StoreInterval == 0 {
		// 🔁 Однократное сохранение при запуске
		if err := ms.SaveMetricsToFile(config); err != nil {
			logger.Error("ошибка при однократном сохранении метрик", zap.Error(err))
		}
		return
	}

	ticker := time.NewTicker(config.StoreInterval)
	defer ticker.Stop()

	logger.Info("запущено периодическое сохранение метрик", zap.Duration("interval", config.StoreInterval))

	for range ticker.C {
		if err := ms.SaveMetricsToFile(config); err != nil {
			logger.Error("ошибка при сохранении метрик", zap.Error(err))
		}
	}
}
