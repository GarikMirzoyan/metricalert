package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GarikMirzoyan/metricalert/internal/database/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPingDBHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDBConn := mocks.NewMockDBConn(ctrl)

	mockDBConn.EXPECT().Ping(gomock.Any()).Return(nil)

	handler := &DBBaseHandler{
		DBConn: mockDBConn,
	}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.PingDBHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestPingDBHandler_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDBConn := mocks.NewMockDBConn(ctrl)

	mockDBConn.EXPECT().Ping(gomock.Any()).Return(fmt.Errorf("database error"))

	handler := &DBBaseHandler{
		DBConn: mockDBConn,
	}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.PingDBHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	assert.Contains(t, w.Body.String(), "Произошла ошибка")
}
