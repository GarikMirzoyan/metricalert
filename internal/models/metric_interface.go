package models

import "github.com/GarikMirzoyan/metricalert/internal/constants"

// Metric defines common behaviour for all metric types.
type Metric interface {
	GetName() string
	GetType() constants.MetricType
	GetValue() any
}
