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
		var batch []models.Metrics

		// Собираем gauge-метрики
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

		// Не отправляем пустой батч
		if len(batch) == 0 {
			continue
		}

		// Отправляем батч с метриками
		metrics.SendBatchMetrics(batch, a.config)
	}
}
