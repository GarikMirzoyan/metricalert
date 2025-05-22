package config

import (
	"flag"
	"os"
	"strings"
	"time"
)

// Структура конфигурации для агента
type Config struct {
	Address        string
	ReportInterval time.Duration
	PollInterval   time.Duration
	Key            string
}

func InitConfig() Config {
	// Значения по умолчанию
	defaultAddress := "localhost:8080"
	defaultReportInterval := 10 * time.Second
	defaultPollInterval := 2 * time.Second
	defaultCryptoKey := ""

	// Читаем флаги командной строки
	address := flag.String("a", defaultAddress, "HTTP server address (without http:// or https://)")
	reportInterval := flag.Int("r", int(defaultReportInterval.Seconds()), "Report interval in seconds")
	pollInterval := flag.Int("p", int(defaultPollInterval.Seconds()), "Poll interval in seconds")
	cryptoKey := flag.String("k", defaultCryptoKey, "Crypto key for hmac")
	flag.Parse()

	// Читаем переменные окружения
	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*address = envAddress
	}

	if envCryptoKey := os.Getenv("KEY"); envCryptoKey != "" {
		*cryptoKey = envCryptoKey
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
		Key:            *cryptoKey,
	}
}
