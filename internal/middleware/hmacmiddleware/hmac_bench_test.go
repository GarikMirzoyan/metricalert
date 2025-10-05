package hmacmiddleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func BenchmarkHMACMiddleware_WithHash(b *testing.B) {
	key := "test-secret"
	m := NewHMACMiddleware(key)

	payload := strings.Repeat("data", 2048)
	handler := m.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Write([]byte(payload))
	}))

	body := []byte(strings.Repeat("req", 1024))
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("HashSHA256", "placeholder")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req.Clone(req.Context()))
	}
}
