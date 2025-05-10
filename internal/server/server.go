package server

import (
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

	if config.DBConnectionString == "" {
		// Работаем с in-memory storage
		handlers := handlers.NewHandlers(storage)
		SetMSRoutes(r, handlers, logger)
	} else {
		// Работаем с БД
		dbConn, err := database.NewDBConnection(config.DBConnectionString)
		if err != nil {
			logger.Error("Error connecting to database", zap.Error(err))
		} else {
			defer dbConn.Close()
			dbHandler := handlers.NewDBHandler(dbConn)
			SetDBRoutes(r, dbHandler, logger)

		}
	}

	server.logger.Info("Starting server", zap.String("address", config.Address))
	if err := http.ListenAndServe(config.Address, r); err != nil {
		server.logger.Error("Error starting server", zap.Error(err))
	}

	if err := storage.SaveMetricsToFile(config); err != nil {
		server.logger.Error("Error saving metrics on shutdown", zap.Error(err))
	}
}

func SetMSRoutes(r *chi.Mux, handlers *handlers.Handlers, logger *zap.Logger) {
	// Добавляем middleware для логирования и сжатия
	r.Use(func(next http.Handler) http.Handler {
		return loggermiddleware.Logger(next, logger)
	})
	r.Use(gzipmiddleware.GzipDecompression) // Разжатие входящих данных
	r.Use(gzipmiddleware.GzipCompression)   // Сжатие исходящих данных

	// Обработчики маршрутов
	r.Post("/update/{type}/{name}/{value}", handlers.UpdateHandler)
	r.Post("/update/", handlers.UpdateHandlerJSON)
	r.Post("/updates/", handlers.BatchMetricsUpdateHandler)
	r.Post("/value/", handlers.GetValueHandlerPost)
	r.Get("/value/{type}/{name}", handlers.GetValueHandler)
	r.Get("/", handlers.RootHandler)

}

func SetDBRoutes(r *chi.Mux, DBHandler *handlers.DBHandler, logger *zap.Logger) {
	// Добавляем middleware для логирования и сжатия
	r.Use(func(next http.Handler) http.Handler {
		return loggermiddleware.Logger(next, logger)
	})
	r.Use(gzipmiddleware.GzipDecompression) // Разжатие входящих данных
	r.Use(gzipmiddleware.GzipCompression)   // Сжатие исходящих данных

	r.Post("/update/{type}/{name}/{value}", DBHandler.UpdateMetricDBHandler)
	r.Post("/update/", DBHandler.UpdateMetricDBHandlerJSON)
	r.Post("/updates/", DBHandler.BatchMetricsUpdateDBHandler)
	r.Get("/value/{type}/{name}", DBHandler.GetMetricValueDBHandler)
	r.Post("/value/", DBHandler.GetValueDBHandlerPost)
	r.Get("/", DBHandler.RootDBHandler)
	r.Get("/ping", DBHandler.PingDBHandler)
}
