package repositories

const (
	queryInsertMetric = `
		INSERT INTO metrics (name, type, value)
		VALUES ($1, $2, $3)
		ON CONFLICT (name) DO UPDATE
		SET value = CASE
			WHEN EXCLUDED.type = 'counter' THEN metrics.value + EXCLUDED.value
			ELSE EXCLUDED.value
		END
	`

	queryInsertSingleMetric = `
		INSERT INTO metrics (name, type, value)
		VALUES ($1, $2, $3::double precision)
		ON CONFLICT (name) DO UPDATE
		SET value = metrics.value + EXCLUDED.value
	`

	querySelectGauge = `
		SELECT value FROM metrics WHERE name = $1 AND type = 'gauge'
	`

	querySelectCounter = `
		SELECT value FROM metrics WHERE name = $1 AND type = 'counter'
	`

	querySelectAllMetrics = `
		SELECT name, type, value FROM metrics
	`
)
