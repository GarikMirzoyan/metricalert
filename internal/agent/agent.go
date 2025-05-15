package agent

import (
	"time"

	"github.com/GarikMirzoyan/metricalert/internal/agent/config"
	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/GarikMirzoyan/metricalert/internal/models"
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

func (a *Agent) Run() {
	go a.startPolling()
	a.startReporting()
}

func (a *Agent) startPolling() {
	ticker := time.NewTicker(a.config.PollInterval)
	for range ticker.C {
		a.pollCount++
	}
}

func (a *Agent) startReporting() {
	ticker := time.NewTicker(a.config.ReportInterval)
	for range ticker.C {
		batch := a.prepareMetricsBatch()
		if len(batch) == 0 {
			continue
		}
		metrics.SendBatchMetrics(batch, a.config)
	}
}

func (a *Agent) prepareMetricsBatch() []models.Metrics {
	var batch []models.Metrics

	// Собираем gauge метрики
	collected := metrics.CollectMetrics()
	for name, value := range collected {
		val := float64(value)
		batch = append(batch, models.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &val,
		})
	}

	// Добавляем PollCount как counter
	delta := int64(a.pollCount)
	batch = append(batch, models.Metrics{
		ID:    "PollCount",
		MType: "counter",
		Delta: &delta,
	})

	return batch
}
