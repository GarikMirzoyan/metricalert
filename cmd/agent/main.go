package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

type Gauge float64
type Counter int64

type Metrics struct {
	ID    string   `json:"id"`              // имя метрики
	MType string   `json:"type"`            // параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // значение метрики в случае передачи gauge
}

type Agent struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
	pollCount      Counter
}

type Config struct {
	Address        string
	ReportInterval time.Duration
	PollInterval   time.Duration
}

func initConfig() Config {
	// Значения по умолчанию
	defaultAddress := "localhost:8080"
	defaultReportInterval := 10 * time.Second
	defaultPollInterval := 2 * time.Second

	// Читаем флаги командной строки
	address := flag.String("a", defaultAddress, "HTTP server address (without http:// or https://)")
	reportInterval := flag.Int("r", int(defaultReportInterval.Seconds()), "Report interval in seconds")
	pollInterval := flag.Int("p", int(defaultPollInterval.Seconds()), "Poll interval in seconds")
	flag.Parse()

	// Читаем переменные окружения
	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*address = envAddress
	}

	if envReportInterval := os.Getenv("REPORT_INTERVAL"); envReportInterval != "" {
		if ri, err := time.ParseDuration(envReportInterval + "s"); err == nil {
			*reportInterval = int(ri.Seconds())
		}
	}

	if envPollInterval := os.Getenv("POLL_INTERVAL"); envPollInterval != "" {
		if pi, err := time.ParseDuration(envPollInterval + "s"); err == nil {
			*pollInterval = int(pi.Seconds())
		}
	}

	finalAddress := *address
	if !strings.HasPrefix(finalAddress, "http://") && !strings.HasPrefix(finalAddress, "https://") {
		finalAddress = "http://" + finalAddress
	}

	return Config{
		Address:        finalAddress,
		ReportInterval: time.Duration(*reportInterval) * time.Second,
		PollInterval:   time.Duration(*pollInterval) * time.Second,
	}
}

func NewAgent(serverAddress string, pollInterval, reportInterval time.Duration) *Agent {
	return &Agent{
		serverAddress:  serverAddress,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		pollCount:      0,
	}
}

func (a *Agent) CollectMetrics() map[string]Gauge {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	metrics := map[string]Gauge{
		"Alloc":         Gauge(memStats.Alloc),
		"BuckHashSys":   Gauge(memStats.BuckHashSys),
		"Frees":         Gauge(memStats.Frees),
		"GCCPUFraction": Gauge(memStats.GCCPUFraction),
		"GCSys":         Gauge(memStats.GCSys),
		"HeapAlloc":     Gauge(memStats.HeapAlloc),
		"HeapIdle":      Gauge(memStats.HeapIdle),
		"HeapInuse":     Gauge(memStats.HeapInuse),
		"HeapObjects":   Gauge(memStats.HeapObjects),
		"HeapReleased":  Gauge(memStats.HeapReleased),
		"HeapSys":       Gauge(memStats.HeapSys),
		"LastGC":        Gauge(memStats.LastGC),
		"Mallocs":       Gauge(memStats.Mallocs),
		"NextGC":        Gauge(memStats.NextGC),
		"PauseTotalNs":  Gauge(memStats.PauseTotalNs),
		"StackInuse":    Gauge(memStats.StackInuse),
		"StackSys":      Gauge(memStats.StackSys),
		"Sys":           Gauge(memStats.Sys),
		"TotalAlloc":    Gauge(memStats.TotalAlloc),
		"RandomValue":   Gauge(rand.Float64()),
	}

	return metrics
}

func (a *Agent) SendMetric(metric Metrics) {
	url := fmt.Sprintf("%s/update/", a.serverAddress)

	body, err := json.Marshal(metric)

	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))

	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()
}

func (a *Agent) Run() {
	tickerPoll := time.NewTicker(a.pollInterval)
	tickerReport := time.NewTicker(a.reportInterval)

	go func() {
		for range tickerPoll.C {
			a.pollCount++
		}
	}()

	for range tickerReport.C {
		// Собираем метрики
		collected := a.CollectMetrics()

		// Отправляем gauge-метрики
		for name, value := range collected {
			val := float64(value)
			metric := Metrics{
				ID:    name,
				MType: "gauge",
				Value: &val,
			}
			a.SendMetric(metric)
		}

		// Отправляем PollCount как counter
		delta := int64(a.pollCount)
		metric := Metrics{
			ID:    "PollCount",
			MType: "counter",
			Delta: &delta,
		}
		a.SendMetric(metric)
	}
}

func main() {
	config := initConfig()

	agent := NewAgent(config.Address, config.PollInterval, config.ReportInterval)
	agent.Run()
}
