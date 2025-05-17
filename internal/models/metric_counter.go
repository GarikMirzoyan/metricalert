package models

import "github.com/GarikMirzoyan/metricalert/internal/constants"

type CounterMetric struct {
	Name  string
	Type  constants.MetricType
	Value int64
}

func (m CounterMetric) GetName() string               { return m.Name }
func (m CounterMetric) GetType() constants.MetricType { return m.Type }
func (m CounterMetric) GetValue() any                 { return m.Value }
