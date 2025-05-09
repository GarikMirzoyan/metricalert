package repositories

import (
	"context"

	"github.com/GarikMirzoyan/metricalert/internal/database"
)

type MetricRepository struct {
	DBConn database.DBConn
}

func NewMetricRepository(DBConn database.DBConn) *MetricRepository {
	MetricRepository := &MetricRepository{DBConn: DBConn}

	return MetricRepository
}

func (ms *MetricRepository) Update(metricType string, metricName string, metricValue string, ctx context.Context) error {
	_, err := ms.DBConn.Exec(ctx, `
        INSERT INTO metrics (name, type, value)
        VALUES ($1, $2, $3::double precision)
        ON CONFLICT (name) DO UPDATE
        SET value = metrics.value + EXCLUDED.value
    `, metricName, metricType, metricValue)

	return err
}

func (ms *MetricRepository) GetGaugeValue(metricName string, ctx context.Context) (float64, error) {
	var value float64
	err := ms.DBConn.QueryRow(ctx, `
		SELECT value FROM metrics WHERE name = $1 AND type = 'gauge'
	`, metricName).Scan(&value)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (ms *MetricRepository) GetCounterValue(metricName string, ctx context.Context) (int64, error) {
	var value int64
	err := ms.DBConn.QueryRow(ctx, `
		SELECT value FROM metrics WHERE name = $1 AND type = 'counter'
	`, metricName).Scan(&value)
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (mr *MetricRepository) GetAllMetrics(ctx context.Context) (map[string]float64, map[string]int64, error) {
	rows, err := mr.DBConn.Query(ctx, `
		SELECT name, type, value FROM metrics
	`)
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
