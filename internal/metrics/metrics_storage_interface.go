package metrics

import (
	"context"

	"github.com/GarikMirzoyan/metricalert/internal/DTO"
	"github.com/GarikMirzoyan/metricalert/internal/models"
)

type MetricStorage interface {
	Update(metric models.Metric, ctx context.Context) error
	UpdateGauge(metric *models.GaugeMetric, ctx context.Context) error
	UpdateCounter(metric *models.CounterMetric, ctx context.Context) error
	UpdateJSON(metric models.Metric, ctx context.Context) (DTO.Metrics, error)
	UpdateBatchJSON(metrics map[string]models.Metric, ctx context.Context) (map[string]DTO.Metrics, error)

	GetValue(metricType, name string, ctx context.Context) (string, error)
	GetGauge(name string, ctx context.Context) (models.GaugeMetric, error)
	GetCounter(name string, ctx context.Context) (models.CounterMetric, error)
	GetJSON(metric models.Metric, ctx context.Context) (DTO.Metrics, error)
	GetAll(ctx context.Context) (map[string]models.GaugeMetric, map[string]models.CounterMetric, error)
}
