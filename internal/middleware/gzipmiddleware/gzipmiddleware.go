package gzipmiddleware

import (
	"compress/gzip"
	"io"
	"net/http"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	Writer io.Writer
}

func (gzw *gzipResponseWriter) Write(p []byte) (int, error) {
	return gzw.Writer.Write(p)
}

func GzipDecompression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Error decompressing data", http.StatusBadRequest)
				return
			}
			defer gz.Close()

			r.Body = gz
		}
		next.ServeHTTP(w, r)
	})
}

// Middleware для сжатия исходящих данных
func GzipCompression(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем, поддерживает ли клиент gzip-сжатие
		if r.Header.Get("Accept-Encoding") == "gzip" {
			// Создаем новый ResponseWriter для сжатия
			w.Header().Set("Content-Encoding", "gzip")
			gz := gzip.NewWriter(w)
			defer gz.Close()

			// Устанавливаем gzipped writer в качестве ResponseWriter
			gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
			next.ServeHTTP(gzw, r)
		} else {
			// Если gzip не поддерживается, просто передаем данные
			next.ServeHTTP(w, r)
		}
	})
}
