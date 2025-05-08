package gzipmiddleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Вспомогательная функция для сжатия данных в gzip
func gzipCompress(data string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(data))
	if err != nil {
		return nil, err
	}
	gz.Close()
	return &buf, nil
}

// Тест для GzipCompression
func TestGzipCompression(t *testing.T) {
	handler := GzipCompression(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	}))

	req := httptest.NewRequest("GET", "http://example.com", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// Проверяем, что ответ сжат
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Errorf("expected Content-Encoding to be gzip, got %v", resp.Header.Get("Content-Encoding"))
	}

	// Распаковываем и проверяем содержимое
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gz.Close()

	unzipped, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to read gzipped content: %v", err)
	}

	expected := "Hello, World!"
	if string(unzipped) != expected {
		t.Errorf("expected response %q, got %q", expected, string(unzipped))
	}
}

// Тест для GzipDecompression
func TestGzipDecompression(t *testing.T) {
	originalBody := "Hello, World!"
	compressedBody, err := gzipCompress(originalBody)
	if err != nil {
		t.Fatalf("failed to gzip compress: %v", err)
	}

	handler := GzipDecompression(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}

		if string(body) != originalBody {
			t.Errorf("expected request body %q, got %q", originalBody, string(body))
		}

		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("POST", "http://example.com", compressedBody)
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()

	// Проверяем, что ответ корректен
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %v", err)
	}

	if string(respBody) != "OK" {
		t.Errorf("expected response body %q, got %q", "OK", string(respBody))
	}
}
