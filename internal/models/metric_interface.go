package models

import "github.com/GarikMirzoyan/metricalert/internal/constants"

type Metric interface {
	GetName() string
	GetType() constants.MetricType
	GetValue() any
}
