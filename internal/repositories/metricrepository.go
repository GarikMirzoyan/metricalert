package repositories

import (
	"context"

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

func (mr *MetricRepository) Update(metricType string, metricName string, metricValue string, ctx context.Context) error {
	_, err := mr.DBConn.Exec(ctx, queryInsertSingleMetric, metricName, metricType, metricValue)

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

func (mr *MetricRepository) GetAllMetrics(ctx context.Context) (map[string]float64, map[string]int64, error) {
	rows, err := mr.DBConn.Query(ctx, querySelectAllMetrics)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	gauges := make(map[string]float64)
	counters := make(map[string]int64)

	for rows.Next() {
		var name, metricType string
		var value float64

		if err := rows.Scan(&name, &metricType, &value); err != nil {
			return nil, nil, err
		}

		switch metricType {
		case "gauge":
			gauges[name] = value
		case "counter":
			counters[name] = int64(value)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return gauges, counters, nil
}

func (mr *MetricRepository) BatchUpdate(metrics []models.Metrics, ctx context.Context) error {
	tx, err := mr.DBConn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, m := range metrics {
		var value interface{}

		switch m.MType {
		case "gauge":
			if m.Value == nil {
				continue
			}
			value = *m.Value

		case "counter":
			if m.Delta == nil {
				continue
			}
			value = float64(*m.Delta)

		default:
			continue
		}

		if _, err := tx.ExecContext(ctx, queryInsertMetric, m.ID, m.MType, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}
