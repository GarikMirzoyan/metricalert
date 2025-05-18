package server

import (
	"net/http"

	"github.com/GarikMirzoyan/metricalert/internal/database"
	"github.com/GarikMirzoyan/metricalert/internal/handlers"
	"github.com/GarikMirzoyan/metricalert/internal/metrics"
	"github.com/GarikMirzoyan/metricalert/internal/middleware/gzipmiddleware"
	"github.com/GarikMirzoyan/metricalert/internal/middleware/loggermiddleware"
	"github.com/GarikMirzoyan/metricalert/internal/repositories"
	"github.com/GarikMirzoyan/metricalert/internal/server/config"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type Server struct {
	storage metrics.MetricStorage
	config  config.Config
	logger  *zap.Logger
}

func NewServer(storage metrics.MetricStorage, logger *zap.Logger, config config.Config) *Server {
	server := &Server{
		storage: storage,
		logger:  logger,
		config:  config,
	}
	return server
}

func Run() {
	r := chi.NewRouter()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	SetMiddlewares(r, logger)

	config := config.InitConfig()

	var storage metrics.MetricStorage

	if config.DBConnectionString == "" {
		// In-memory storage
		memStorage := metrics.NewMemStorage()

		if err := memStorage.LoadMetricsFromFile(config); err != nil {
			logger.Error("Error loading metrics", zap.Error(err))
		}

		go memStorage.StartMetricSaving(config, logger)

		if err := memStorage.SaveMetricsToFile(config); err != nil {
			logger.Error("Error saving metrics on shutdown", zap.Error(err))
		}

		storage = memStorage
	} else {
		// Подключение к базе
		dbConn, err := database.NewDBConnection(config.DBConnectionString)
		if err != nil {
			logger.Fatal("Error connecting to database", zap.Error(err))
		}
		defer dbConn.Close()

		if err := dbConn.RunMigrations(); err != nil {
			logger.Fatal("Migration error", zap.Error(err))
		}

		repo := repositories.NewMetricRepository(dbConn)
		dbStorage := metrics.NewDBStorage(repo)
		storage = dbStorage

		dbBaseHandlers := handlers.NewDBBaseHandlers(dbConn)
		SetDBRoutes(r, dbBaseHandlers)
	}

	server := NewServer(storage, logger, config)

	handlers := handlers.NewHandlers(storage)

	SetMetricRoutes(r, handlers)

	// // Загружаем метрики, если указано
	// if err := storage.LoadMetricsFromFile(server.config); err != nil {
	// 	logger.Error("Error loading metrics", zap.Error(err))
	// }

	// // Запускаем сохранение метрик
	// go storage.StartMetricSaving(server.config, server.logger)

	server.logger.Info("Starting server", zap.String("address", config.Address))
	if err := http.ListenAndServe(config.Address, r); err != nil {
		server.logger.Error("Error starting server", zap.Error(err))
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

func SetMetricRoutes(r *chi.Mux, handlers *handlers.Handler) {
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
