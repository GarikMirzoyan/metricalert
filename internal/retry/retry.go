package retry

import (
	"errors"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

func WithBackoff(action func() error) error {
	delays := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}
	var lastErr error

	for i, delay := range delays {
		err := action()
		if err == nil {
			return nil
		}
		if !IsRetriableError(err) {
			return err
		}
		lastErr = err
		log.Printf("попытка %d неудачна: %v — повтор через %s", i+1, err, delay)
		time.Sleep(delay)
	}
	return lastErr
}

func IsRetriableError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "connection reset") {
		return true
	}
	return false
}
