package hmacmiddleware

import (
	"bytes"
	"io"
	"net/http"
	"sync"

	"github.com/GarikMirzoyan/metricalert/internal/security"
)

// HMACMiddleware validates request body using HMAC-SHA256 and signs responses.
type HMACMiddleware struct {
	Key []byte
}

// NewHMACMiddleware creates middleware with the provided HMAC key.
func NewHMACMiddleware(key string) *HMACMiddleware {
	return &HMACMiddleware{
		Key: []byte(key),
	}
}

// Middleware validates incoming request HMAC (when provided) and
// attaches a response HMAC header "HashSHA256" for the response body.
func (h *HMACMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHash := r.Header.Get("HashSHA256")

		// Если подпись пришла, но у сервера нет ключа — отклоняем
		if receivedHash != "" && len(h.Key) == 0 {
			http.Error(w, "HMAC signature provided but server has no key", http.StatusBadRequest)
			return
		}

		var err error

		if receivedHash != "" {
			// Читаем тело только если нужна проверка
			buf := requestBufPool.Get().(*bytes.Buffer)
			buf.Reset()

			defer func() {
				buf.Reset()
				requestBufPool.Put(buf)
			}()

			if _, err = io.Copy(buf, r.Body); err != nil {
				http.Error(w, "cannot read body", http.StatusBadRequest)
				return
			}
			r.Body.Close()

			// Восстанавливаем тело запроса для следующих обработчиков
			r.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))

			// Проверка подписи
			expectedHash := security.ComputeHMACSHA256(buf.Bytes(), h.Key)
			if receivedHash != expectedHash {
				http.Error(w, "invalid HMAC signature", http.StatusBadRequest)
				return
			}

			// Оборачиваем writer для подписи ответа
			respBuf := responseBufPool.Get().(*bytes.Buffer)
			respBuf.Reset()
			rec := &responseWriterWithHash{
				ResponseWriter: w,
				key:            h.Key,
				buf:            respBuf,
			}

			next.ServeHTTP(rec, r)

			hash := security.ComputeHMACSHA256(rec.buf.Bytes(), h.Key)
			rec.Header().Set("HashSHA256", hash)

			if rec.statusCode == 0 {
				rec.statusCode = http.StatusOK
			}
			rec.ResponseWriter.WriteHeader(rec.statusCode)
			_, _ = rec.ResponseWriter.Write(rec.buf.Bytes())

			rec.buf.Reset()
			responseBufPool.Put(rec.buf)
		} else {
			// Без ключа — обычный ответ
			next.ServeHTTP(w, r)
		}
	})
}

type responseWriterWithHash struct {
	http.ResponseWriter
	key        []byte
	buf        *bytes.Buffer
	statusCode int
}

func (r *responseWriterWithHash) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseWriterWithHash) Write(b []byte) (int, error) {
	return r.buf.Write(b)
}

var requestBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}
var responseBufPool = sync.Pool{New: func() any { return new(bytes.Buffer) }}
