package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/GarikMirzoyan/metricalert/internal/metrics"
)

type Agent struct {
	serverAddress  string
	pollInterval   time.Duration
	reportInterval time.Duration
	pollCount      metrics.Counter
}

func NewAgent(serverAddress string, pollInterval, reportInterval time.Duration) *Agent {
	return &Agent{
		serverAddress:  serverAddress,
		pollInterval:   pollInterval,
		reportInterval: reportInterval,
		pollCount:      0,
	}
}

// Измененная функция отправки метрик с поддержкой gzip
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
		collected := metrics.CollectMetrics()

		// Отправляем gauge-метрики
		for name, value := range collected {
			val := float64(value)
			metric := metrics.Metrics{
				ID:    name,
				MType: "gauge",
				Value: &val,
			}
			a.SendMetric(metric)
		}

		// Отправляем PollCount как counter
		delta := int64(a.pollCount)
		metric := metrics.Metrics{
			ID:    "PollCount",
			MType: "counter",
			Delta: &delta,
		}
		a.SendMetric(metric)
	}
}

func (a *Agent) SendMetric(metric metrics.Metrics) {
	url := fmt.Sprintf("%s/update/", a.serverAddress)

	body, err := json.Marshal(metric)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %v\n", err)
		return
	}

	// Сжимаем данные перед отправкой
	compressedBody, err := compressGzip(body)
	if err != nil {
		fmt.Printf("Error compressing data: %v\n", err)
		return
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(compressedBody))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	// Устанавливаем заголовки для gzip
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()
}

// Функция для сжатия данных в формате gzip
func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	_, err := gzipWriter.Write(data)
	if err != nil {
		return nil, err
	}
	gzipWriter.Close()
	return buf.Bytes(), nil
}
