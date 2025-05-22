package hmacmiddleware

import (
	"bytes"
	"io"
	"net/http"

	"github.com/GarikMirzoyan/metricalert/internal/security"
)

type HMACMiddleware struct {
	Key []byte
}

func NewHMACMiddleware(key string) *HMACMiddleware {
	return &HMACMiddleware{
		Key: []byte(key),
	}
}

func (h *HMACMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(h.Key) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read body", http.StatusBadRequest)
			return
		}
		r.Body.Close()

		r.Body = io.NopCloser(bytes.NewBuffer(body))

		expectedHash := security.ComputeHMACSHA256(body, h.Key)
		receivedHash := r.Header.Get("HashSHA256")

		if receivedHash != expectedHash {
			http.Error(w, "invalid HMAC signature", http.StatusBadRequest)
			return
		}

		rec := &responseWriterWithHash{
			ResponseWriter: w,
			key:            h.Key,
			buf:            &bytes.Buffer{},
		}

		next.ServeHTTP(rec, r)

		hash := security.ComputeHMACSHA256(rec.buf.Bytes(), h.Key)
		rec.Header().Set("HashSHA256", hash)

		rec.ResponseWriter.WriteHeader(rec.statusCode)
		rec.ResponseWriter.Write(rec.buf.Bytes())
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
