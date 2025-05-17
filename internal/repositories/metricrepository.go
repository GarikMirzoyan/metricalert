package repositories

import (
	"context"

	"github.com/GarikMirzoyan/metricalert/internal/constants"
	"github.com/GarikMirzoyan/metricalert/internal/database"
	"github.com/GarikMirzoyan/metricalert/internal/models"
)

type MetricRepository struct {
	DBConn database.DBConn
}

func NewMetricRepository(DBConn database.DBConn) *MetricRepository {
	MetricRepository := &MetricRepository{DBConn: DBConn}

	return MetricRepository
}

func (mr *MetricRepository) Update(metric models.Metric, ctx context.Context) error {
	_, err := mr.DBConn.Exec(ctx, queryInsertSingleMetric, metric.GetName(), metric.GetType(), metric.GetValue())

	return err
}

func (mr *MetricRepository) GetGaugeValue(metricName string, ctx context.Context) (float64, error) {
	var value float64
	err := mr.DBConn.QueryRow(ctx, querySelectGauge, metricName).Scan(&value)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (mr *MetricRepository) GetCounterValue(metricName string, ctx context.Context) (int64, error) {
	var fvalue float64
	err := mr.DBConn.QueryRow(ctx, querySelectCounter, metricName).Scan(&fvalue)
	if err != nil {
		return 0, err
	}
	return int64(fvalue), nil
}

func (mr *MetricRepository) GetAllMetrics(ctx context.Context) (map[string]models.GaugeMetric, map[string]models.CounterMetric, error) {
	rows, err := mr.DBConn.Query(ctx, querySelectAllMetrics)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	gauges := make(map[string]models.GaugeMetric)
	counters := make(map[string]models.CounterMetric)

	for rows.Next() {
		var name string
		var metricType constants.MetricType
		var value float64

		if err := rows.Scan(&name, &metricType, &value); err != nil {
			return nil, nil, err
		}

		switch metricType {
		case constants.GaugeName:
			gauges[name] = models.GaugeMetric{
				Name:  name,
				Type:  constants.GaugeName,
				Value: value,
			}
		case constants.CounterName:
			counters[name] = models.CounterMetric{
				Name:  name,
				Type:  constants.CounterName,
				Value: int64(value),
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return gauges, counters, nil
}

func (mr *MetricRepository) BatchUpdate(metrics []models.Metric, ctx context.Context) error {
	tx, err := mr.DBConn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, m := range metrics {
		name := m.GetName()
		typ := m.GetType()
		val := m.GetValue()

		if val == nil {
			continue
		}

		var value float64

		switch typ {
		case constants.GaugeName:
			v, ok := val.(float64)
			if !ok {
				continue
			}
			value = v

		case constants.CounterName:
			v, ok := val.(int64)
			if !ok {
				continue
			}
			value = float64(v)

		default:
			continue
		}

		if _, err := tx.ExecContext(ctx, queryInsertMetric, name, typ, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}
