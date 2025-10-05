package models

import "github.com/GarikMirzoyan/metricalert/internal/constants"

// CounterMetric represents an integer metric.
type CounterMetric struct {
	Name  string
	Type  constants.MetricType
	Value int64
}

func (m CounterMetric) GetName() string               { return m.Name }
func (m CounterMetric) GetType() constants.MetricType { return m.Type }
func (m CounterMetric) GetValue() any                 { return m.Value }
