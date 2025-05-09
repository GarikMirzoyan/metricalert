package config

import (
	"flag"
	"os"
	"time"
)

// Структура конфигурации для сервера
type Config struct {
	StoreInterval      time.Duration
	FileStoragePath    string
	Restore            bool
	Address            string
	DBConnectionString string
}

func InitConfig() Config {
	defaultStoreInterval := 30 * time.Second
	defaultFileStoragePath := "../../internal/metrics/data/metrics.json"
	defaultRestore := true
	//"postgres://mirzoangarikaregovic@localhost:5432/metricalert"
	defaultDBConnectionString := ""

	defaultAddress := "localhost:8080"
	address := flag.String("a", defaultAddress, "HTTP server address (without http:// or https://)")
	storeInterval := flag.Int("i", int(defaultStoreInterval.Seconds()), "Interval for saving metrics (in seconds)")
	fileStoragePath := flag.String("f", defaultFileStoragePath, "Path to file where metrics will be saved")
	restore := flag.Bool("r", defaultRestore, "Restore metrics from file on start (true/false)")
	DBConnectionString := flag.String("d", defaultDBConnectionString, "DB connction string")
	flag.Parse()

	if envAddress := os.Getenv("ADDRESS"); envAddress != "" {
		*address = envAddress
	}

	if envDBDSN := os.Getenv("DATABASE_DSN"); envDBDSN != "" {
		*DBConnectionString = envDBDSN
	}

	// Чтение из переменных окружения
	if envStoreInterval := os.Getenv("STORE_INTERVAL"); envStoreInterval != "" {
		if si, err := time.ParseDuration(envStoreInterval + "s"); err == nil {
			*storeInterval = int(si.Seconds())
		}
	}

	if envFileStoragePath := os.Getenv("FILE_STORAGE_PATH"); envFileStoragePath != "" {
		*fileStoragePath = envFileStoragePath
	}

	if envRestore := os.Getenv("RESTORE"); envRestore != "" {
		*restore = envRestore == "true"
	}

	return Config{
		StoreInterval:      time.Duration(*storeInterval) * time.Second,
		FileStoragePath:    *fileStoragePath,
		Restore:            *restore,
		Address:            *address,
		DBConnectionString: *DBConnectionString,
	}
}
