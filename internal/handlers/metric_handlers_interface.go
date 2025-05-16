package handlers

import "net/http"

type MetricsHandlers interface {
	UpdateHandler(w http.ResponseWriter, r *http.Request)
	UpdateHandlerJSON(w http.ResponseWriter, r *http.Request)
	GetValueHandler(w http.ResponseWriter, r *http.Request)
	GetValueHandlerJSON(w http.ResponseWriter, r *http.Request)
	RootHandler(w http.ResponseWriter, r *http.Request)
	BatchMetricsUpdateHandler(w http.ResponseWriter, r *http.Request)
}
