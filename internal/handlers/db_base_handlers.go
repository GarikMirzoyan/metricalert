package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/GarikMirzoyan/metricalert/internal/database"
)

type DBBaseHandler struct {
	DBConn database.DBConn
}

func NewDBBaseHandlers(DBConn database.DBConn) *DBBaseHandler {
	DBBaseHandler := &DBBaseHandler{DBConn: DBConn}

	return DBBaseHandler
}

func (h *DBBaseHandler) PingDBHandler(w http.ResponseWriter, r *http.Request) {
	err := h.DBConn.Ping(context.Background())

	if err != nil {
		http.Error(w, fmt.Sprintf("Произошла ошибка: %v", err), http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}
