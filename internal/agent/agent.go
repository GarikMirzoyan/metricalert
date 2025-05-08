package agent

import (
	"time"

	"github.com/GarikMirzoyan/metricalert/internal/agent/config"
	"github.com/GarikMirzoyan/metricalert/internal/metrics"
)

type Agent struct {
	config    config.Config
	pollCount metrics.Counter
}

func NewAgent(config config.Config) *Agent {
	return &Agent{
		config:    config,
		pollCount: 0,
	}
}

// Измененная функция отправки метрик с поддержкой gzip
func (a *Agent) Run() {
	tickerPoll := time.NewTicker(a.config.PollInterval)
	tickerReport := time.NewTicker(a.config.ReportInterval)

	go func() {
		for range tickerPoll.C {
			a.pollCount++
		}
	}()

	for range tickerReport.C {
		// Собираем метрики
		collected := metrics.CollectMetrics()

		// Отправляем gauge-метрики
		for name, value := range collected {
			val := float64(value)
			metric := metrics.Metrics{
				ID:    name,
				MType: "gauge",
				Value: &val,
			}
			metrics.SendMetric(metric, a.config)
		}

		// Отправляем PollCount как counter
		delta := int64(a.pollCount)
		metric := metrics.Metrics{
			ID:    "PollCount",
			MType: "counter",
			Delta: &delta,
		}
		metrics.SendMetric(metric, a.config)
	}
}
