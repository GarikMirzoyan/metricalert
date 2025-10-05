package constants

// MetricType is a string alias listing supported metric kinds.
type MetricType string

const (
	GaugeName   MetricType = "gauge"
	CounterName MetricType = "counter"
)
