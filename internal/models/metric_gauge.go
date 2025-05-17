package models

import "github.com/GarikMirzoyan/metricalert/internal/constants"

type GaugeMetric struct {
	Name  string
	Type  constants.MetricType
	Value float64
}

func (m GaugeMetric) GetName() string               { return m.Name }
func (m GaugeMetric) GetType() constants.MetricType { return m.Type }
func (m GaugeMetric) GetValue() any                 { return m.Value }
