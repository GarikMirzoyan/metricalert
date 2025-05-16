package server

import (
	"log"
	"net/http"

	"github.com/GarikMirzoyan/metricalert/internal/database"
	"github.com/GarikMirzoyan/metricalert/internal/handlers"
	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/GarikMirzoyan/metricalert/internal/middleware/gzipmiddleware"
	"github.com/GarikMirzoyan/metricalert/internal/middleware/loggermiddleware"
	"github.com/GarikMirzoyan/metricalert/internal/server/config"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type Server struct {
	storage *metrics.MemStorage
	config  config.Config
	logger  *zap.Logger
}

func NewServer(storage *metrics.MemStorage, logger *zap.Logger, config config.Config) *Server {
	server := &Server{
		storage: storage,
		logger:  logger,
		config:  config,
	}
	return server
}

func Run() {
	// Настройка логирования с использованием zap
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	config := config.InitConfig()
	storage := metrics.NewMemStorage()

	server := NewServer(storage, logger, config)

	r := chi.NewRouter()

	// Загружаем метрики, если указано
	if err := storage.LoadMetricsFromFile(server.config); err != nil {
		logger.Error("Error loading metrics", zap.Error(err))
	}

	// Запускаем сохранение метрик
	go storage.StartMetricSaving(server.config, server.logger)

	SetMiddlewares(r, logger)
	if config.DBConnectionString == "" {
		// Работаем с in-memory storage
		handlers := handlers.NewMemHandlers(storage)
		SetMetricRoutes(r, handlers)
	} else {
		dbConn, err := database.NewDBConnection(config.DBConnectionString)
		if err != nil {
			logger.Fatal("Error connecting to database", zap.Error(err)) // или log.Fatalf(...)
		}
		defer dbConn.Close()

		// Прогон миграций
		if err := dbConn.RunMigrations(); err != nil {
			log.Fatalf("Migration error: %v", err)
		}

		dbBaseHandlers := handlers.NewDBBaseHandlers(dbConn)
		// Подключение хендлеров и маршрутов
		handlers := handlers.NewDBHandlers(dbConn)
		SetMetricRoutes(r, handlers)
		SetDBRoutes(r, dbBaseHandlers)
	}

	server.logger.Info("Starting server", zap.String("address", config.Address))
	if err := http.ListenAndServe(config.Address, r); err != nil {
		server.logger.Error("Error starting server", zap.Error(err))
	}

	if err := storage.SaveMetricsToFile(config); err != nil {
		server.logger.Error("Error saving metrics on shutdown", zap.Error(err))
	}
}

func SetMiddlewares(r *chi.Mux, logger *zap.Logger) {
	// Добавляем middleware для логирования и сжатия
	r.Use(func(next http.Handler) http.Handler {
		return loggermiddleware.Logger(next, logger)
	})
	r.Use(gzipmiddleware.GzipDecompression) // Разжатие входящих данных
	r.Use(gzipmiddleware.GzipCompression)   // Сжатие исходящих данных
}

func SetMetricRoutes(r *chi.Mux, handlers handlers.MetricsHandlers) {
	r.Post("/update/{type}/{name}/{value}", handlers.UpdateHandler)
	r.Post("/update/", handlers.UpdateHandlerJSON)
	r.Post("/updates/", handlers.BatchMetricsUpdateHandler)
	r.Get("/value/{type}/{name}", handlers.GetValueHandler)
	r.Post("/value/", handlers.GetValueHandlerJSON)
	r.Get("/", handlers.RootHandler)
}

func SetDBRoutes(r *chi.Mux, handlers *handlers.DBBaseHandler) {
	r.Get("/ping", handlers.PingDBHandler)
}
