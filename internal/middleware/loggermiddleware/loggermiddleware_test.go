package loggermiddleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// вспомогательная функция для создания тестового логгера, который пишет в строку
func newTestLogger(t *testing.T) (*zap.Logger, *strings.Builder) {
	var logs strings.Builder

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "" // отключим время для читаемости
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.AddSync(&logs),
		zap.DebugLevel,
	)

	logger := zap.New(core)
	return logger, &logs
}

func TestLoggerMiddleware_BasicFlow(t *testing.T) {
	logger, logs := newTestLogger(t)

	// тестовый хендлер
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot) // 418 :)
		w.Write([]byte("Hello, world!"))
	})

	// оборачиваем middleware
	handlerToTest := Logger(testHandler, logger)

	// создаем запрос и записываем ответ
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handlerToTest.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// проверка статуса
	if resp.StatusCode != http.StatusTeapot {
		t.Errorf("Expected status %d, got %d", http.StatusTeapot, resp.StatusCode)
	}

	// проверка тела
	body := w.Body.String()
	if body != "Hello, world!" {
		t.Errorf("Unexpected body: %s", body)
	}

	// проверка что логгер что-то записал
	logOutput := logs.String()
	if !strings.Contains(logOutput, "Handled request") {
		t.Errorf("Expected log to contain 'Handled request', got: %s", logOutput)
	}

	if !strings.Contains(logOutput, "method") || !strings.Contains(logOutput, "GET") {
		t.Errorf("Expected log to contain method info, got: %s", logOutput)
	}

	if !strings.Contains(logOutput, "status") || !strings.Contains(logOutput, "418") {
		t.Errorf("Expected log to contain status code 418, got: %s", logOutput)
	}
}
