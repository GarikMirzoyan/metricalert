package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GarikMirzoyan/metricalert/internal/database/mocks"
	"github.com/go-chi/chi"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) { return 0, nil }
func (r *mockResult) RowsAffected() (int64, error) { return 1, nil }

func TestUpdateHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDBConn := mocks.NewMockDBConn(ctrl)

	mockDBConn.
		EXPECT().
		Exec(gomock.Any(), gomock.Any(), "testCounter", "counter", "10").
		Return(&mockResult{}, nil)

	handler := &DBHandler{
		DBConn: mockDBConn,
	}

	req := httptest.NewRequest(http.MethodPost, "/update/counter/testCounter/10", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "counter")
	rctx.URLParams.Add("name", "testCounter")
	rctx.URLParams.Add("value", "10")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateHandler_Error(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDBConn := mocks.NewMockDBConn(ctrl)

	mockDBConn.
		EXPECT().
		Exec(gomock.Any(), gomock.Any(), "testGauge", "gauge", "42.42").
		Return(&mockResult{}, fmt.Errorf("some DB error"))

	handler := &DBHandler{
		DBConn: mockDBConn,
	}

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/testGauge/42.42", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("type", "gauge")
	rctx.URLParams.Add("name", "testGauge")
	rctx.URLParams.Add("value", "42.42")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	w := httptest.NewRecorder()

	handler.UpdateHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Произошла ошибка")
}
