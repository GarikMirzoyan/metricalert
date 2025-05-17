package metrics

import (
	"context"

	dto "github.com/GarikMirzoyan/metricalert/internal/DTO"
	"github.com/GarikMirzoyan/metricalert/internal/models"
)

type MetricStorage interface {
	Update(metric models.Metric, ctx context.Context) error
	UpdateGauge(metric *models.GaugeMetric, ctx context.Context) error
	UpdateCounter(metric *models.CounterMetric, ctx context.Context) error
	UpdateJSON(metric models.Metric, ctx context.Context) (dto.Metrics, error)
	UpdateBatchJSON(metrics []models.Metric, ctx context.Context) (map[string]dto.Metrics, error)

	GetValue(metricType, name string, ctx context.Context) (string, error)
	GetGauge(name string, ctx context.Context) (models.GaugeMetric, error)
	GetCounter(name string, ctx context.Context) (models.CounterMetric, error)
	GetJSON(metric models.Metric, ctx context.Context) (dto.Metrics, error)
	GetAll(ctx context.Context) (map[string]models.GaugeMetric, map[string]models.CounterMetric, error)
}
