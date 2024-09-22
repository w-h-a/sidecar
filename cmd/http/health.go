package http

import "net/http"

type HealthHandler interface {
	Check(w http.ResponseWriter, r *http.Request)
}

type healthHandler struct {
}

func (h *healthHandler) Check(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func NewHealthHandler() HealthHandler {
	return &healthHandler{}
}
