package agent

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	dto "github.com/GarikMirzoyan/metricalert/internal/DTO"
	"github.com/GarikMirzoyan/metricalert/internal/agent/config"
	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
)

var gopsutilMetrics = make(map[string]float64)
var gopsutilMu sync.Mutex

type Agent struct {
	config    config.Config
	pollCount metrics.Counter
	jobs      chan MetricJob
	done      chan struct{}
}

type MetricJob struct {
	Batch []dto.Metrics
}

func NewAgent(config config.Config) *Agent {
	return &Agent{
		config:    config,
		pollCount: 0,
		jobs:      make(chan MetricJob, config.RateLimit*2),
		done:      make(chan struct{}),
	}
}

func (a *Agent) Run() {
	for i := 0; i < a.config.RateLimit; i++ {
		go a.worker()
	}

	go a.startPolling()
	go a.startGopsutilPolling()
	go a.startBatching()
	log.Printf("Активных горутин: %d", runtime.NumGoroutine())
	<-a.done
}

func (a *Agent) startPolling() {
	ticker := time.NewTicker(a.config.PollInterval)
	for range ticker.C {
		a.pollCount++
	}
}

func (a *Agent) startBatching() {
	ticker := time.NewTicker(a.config.ReportInterval)
	for range ticker.C {
		batch := a.prepareMetricsBatch()
		if len(batch) > 0 {
			a.jobs <- MetricJob{Batch: batch}
		}
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

func (a *Agent) prepareMetricsBatch() []dto.Metrics {
	var batch []dto.Metrics

	collected := metrics.CollectMetrics()
	for name, value := range collected {
		val := float64(value)
		batch = append(batch, dto.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &val,
		})
	}

	gopsutilMu.Lock()
	for name, val := range gopsutilMetrics {
		v := val
		batch = append(batch, dto.Metrics{
			ID:    name,
			MType: "gauge",
			Value: &v,
		})
	}
	gopsutilMu.Unlock()

	// Counter PollCount
	delta := int64(a.pollCount)
	batch = append(batch, dto.Metrics{
		ID:    "PollCount",
		MType: "counter",
		Delta: &delta,
	})

	return batch
}

func (a *Agent) worker() {
	for job := range a.jobs {
		metrics.SendBatchMetrics(job.Batch, a.config)
	}
}

func (a *Agent) startGopsutilPolling() {
	ticker := time.NewTicker(a.config.PollInterval)
	for range ticker.C {
		memStats, err := mem.VirtualMemory()
		if err == nil {
			gopsutilMu.Lock()
			gopsutilMetrics["TotalMemory"] = float64(memStats.Total)
			gopsutilMetrics["FreeMemory"] = float64(memStats.Free)
			gopsutilMu.Unlock()
		}

		cpuStats, err := cpu.Percent(0, true)
		if err == nil {
			gopsutilMu.Lock()
			for i, v := range cpuStats {
				gopsutilMetrics[fmt.Sprintf("CPUutilization1_CPU%d", i)] = v
			}
			gopsutilMu.Unlock()
		}
	}
}
