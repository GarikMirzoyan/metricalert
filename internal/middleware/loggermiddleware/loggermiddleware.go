package loggermiddleware

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type statusWriter struct {
	http.ResponseWriter
	status int
	size   int64
}

func (ww *statusWriter) WriteHeader(status int) {
	ww.status = status
	ww.ResponseWriter.WriteHeader(status)
}

func (ww *statusWriter) Write(p []byte) (int, error) {
	size, err := ww.ResponseWriter.Write(p)
	ww.size += int64(size)
	return size, err
}

// Middleware для логирования запросов и ответов
func Logger(next http.Handler, logger *zap.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Подготовим кастомный ResponseWriter для получения информации о статусе и размере ответа
		ww := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		// Логируем информацию о запросе и ответе
		logger.Info("Handled request",
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.Duration("duration", duration),
			zap.Int("status", ww.status),
			zap.Int64("response_size", ww.size),
		)
	})
}
