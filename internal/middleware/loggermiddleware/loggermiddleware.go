package loggermiddleware

import (
	"bytes"
	"compress/gzip"
	"io"
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

		var bodyBytes []byte
		if r.Body != nil {
			contentEncoding := r.Header.Get("Content-Encoding")
			switch contentEncoding {
			case "gzip":
				gzipReader, err := gzip.NewReader(r.Body)
				if err != nil {
					logger.Warn("could not create gzip reader", zap.Error(err))
				} else {
					defer gzipReader.Close()
					bodyBytes, err = io.ReadAll(gzipReader)
					if err != nil {
						logger.Warn("could not read gzipped request body", zap.Error(err))
					}
				}
			default:
				b, err := io.ReadAll(r.Body)
				if err != nil {
					logger.Warn("could not read request body", zap.Error(err))
				} else {
					bodyBytes = b
				}
			}
		}

		// Восстанавливаем тело запроса для обработки хендлером
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Обернутый ResponseWriter
		ww := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(ww, r)

		duration := time.Since(start)

		logger.Info("Handled request",
			zap.String("method", r.Method),
			zap.String("uri", r.RequestURI),
			zap.String("body", string(bodyBytes)),
			zap.Duration("duration", duration),
			zap.Int("status", ww.status),
			zap.Int64("response_size", ww.size),
		)
	})
}
