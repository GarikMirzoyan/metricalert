package main

import (
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
		"Alloc":       Gauge(memStats.Alloc),
		"RandomValue": Gauge(rand.Float64()),
	}

	return metrics
}

func (a *Agent) SendMetric(metricType, metricName string, value interface{}) {
	url := fmt.Sprintf("%s/update/%s/%s/%v", a.serverAddress, metricType, metricName, value)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "text/plain")

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

	metrics := make(map[string]Gauge)

	go func() {
		for range tickerPoll.C {
			a.pollCount++
			metrics = a.CollectMetrics()
			metrics["PollCount"] = Gauge(a.pollCount)
		}
	}()

	for range tickerReport.C {
		for name, value := range metrics {
			a.SendMetric("gauge", name, value)
		}
		a.SendMetric("counter", "PollCount", a.pollCount)
	}
}

func main() {
	config := initConfig()

	agent := NewAgent(config.Address, config.PollInterval, config.ReportInterval)
	agent.Run()
}
