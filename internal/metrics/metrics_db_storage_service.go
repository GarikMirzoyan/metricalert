package metrics

import (
	"context"
	"errors"
	"fmt"

	dto "github.com/GarikMirzoyan/metricalert/internal/DTO"
	"github.com/GarikMirzoyan/metricalert/internal/constants"
	"github.com/GarikMirzoyan/metricalert/internal/models"
	"github.com/GarikMirzoyan/metricalert/internal/repositories"
)

type DBStorage struct {
	metricRepository *repositories.MetricRepository
}

func NewDBStorage(metricRepository *repositories.MetricRepository) *DBStorage {
	return &DBStorage{
		metricRepository: metricRepository,
	}
}

func (ms *DBStorage) Update(metric models.Metric, ctx context.Context) error {
	switch m := metric.(type) {
	case *models.GaugeMetric:
		return ms.UpdateGauge(m, ctx)
	case *models.CounterMetric:
		return ms.UpdateCounter(m, ctx)
	default:
		return fmt.Errorf("invalid metric type")
	}
}

func (ms *DBStorage) UpdateJSON(metric models.Metric, ctx context.Context) (dto.Metrics, error) {
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
		value, ok := metric.GetValue().(float64)
		if !ok {
			return dto.Metrics{}, fmt.Errorf("invalid type assertion: expected *float64")
		}

		err := ms.metricRepository.Update(m, ctx)
		if err != nil {
			return dto.Metrics{}, err
		}

		response.Value = &value

	case *models.CounterMetric:
		delta, ok := metric.GetValue().(int64)
		if !ok {
			return dto.Metrics{}, fmt.Errorf("invalid type assertion: expected *int64")
		}

		err := ms.metricRepository.Update(m, ctx)
		if err != nil {
			return dto.Metrics{}, err
		}

		response.Delta = &delta

	default:
		return dto.Metrics{}, ErrInvalidMetricType
	}

	return response, nil
}

func (ms *DBStorage) GetValue(metricType, metricName string, ctx context.Context) (string, error) {
	switch constants.MetricType(metricType) {
	case constants.GaugeName:
		value, err := ms.metricRepository.GetGaugeValue(metricName, ctx)
		if err != nil {
			return "", fmt.Errorf("произошла ошибка при получении Gauge: %w", err)
		}
		return fmt.Sprintf("%v", value), nil

	case constants.CounterName:
		value, err := ms.metricRepository.GetCounterValue(metricName, ctx)
		if err != nil {
			return "", fmt.Errorf("произошла ошибка при получении Counter: %w", err)
		}
		return fmt.Sprintf("%v", value), nil

	default:
		return "", ErrInvalidMetricType
	}
}

func (ms *DBStorage) GetJSON(metric models.Metric, ctx context.Context) (dto.Metrics, error) {
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

func (ms *DBStorage) UpdateGauge(metric *models.GaugeMetric, ctx context.Context) error {
	if metric.Name == "" {
		return fmt.Errorf("gauge metric name is empty")
	}
	return ms.metricRepository.Update(metric, ctx)
}

func (ms *DBStorage) UpdateCounter(metric *models.CounterMetric, ctx context.Context) error {
	if metric.Name == "" {
		return fmt.Errorf("counter metric name is empty")
	}

	currentValue, err := ms.metricRepository.GetCounterValue(metric.Name, ctx)
	if err != nil && !errors.Is(err, ErrMetricNotFound) {
		return err
	}

	newValue := currentValue + metric.Value
	metric.Value = newValue

	return ms.metricRepository.Update(metric, ctx)
}

func (ms *DBStorage) GetGauge(name string, ctx context.Context) (models.GaugeMetric, error) {
	val, err := ms.metricRepository.GetGaugeValue(name, ctx)
	if err != nil {
		return models.GaugeMetric{}, fmt.Errorf("get gauge failed: %w", ErrMetricNotFound)
	}

	return models.GaugeMetric{
		Name:  name,
		Type:  constants.GaugeName,
		Value: val,
	}, nil
}

func (ms *DBStorage) GetCounter(name string, ctx context.Context) (models.CounterMetric, error) {
	val, err := ms.metricRepository.GetCounterValue(name, ctx)
	if err != nil {
		return models.CounterMetric{}, fmt.Errorf("get counter failed: %w", ErrMetricNotFound)
	}

	return models.CounterMetric{
		Name:  name,
		Type:  constants.CounterName,
		Value: val,
	}, nil
}

func (ms *DBStorage) GetAll(ctx context.Context) (map[string]models.GaugeMetric, map[string]models.CounterMetric, error) {
	gauges, counters, err := ms.metricRepository.GetAllMetrics(ctx)
	if err != nil {
		return nil, nil, err
	}

	return gauges, counters, nil
}

func (ms *DBStorage) UpdateBatchJSON(metrics map[string]models.Metric, ctx context.Context) (map[string]dto.Metrics, error) {
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
