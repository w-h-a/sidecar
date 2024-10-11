package http

import (
	"net/http"
	"strconv"

	"github.com/w-h-a/pkg/telemetry/trace"
	"github.com/w-h-a/pkg/utils/errorutils"
	"github.com/w-h-a/pkg/utils/httputils"
)

type HealthHandler interface {
	Check(w http.ResponseWriter, r *http.Request)
	Trace(w http.ResponseWriter, r *http.Request)
}

type healthHandler struct {
	tracer trace.Trace
}

func (h *healthHandler) Check(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func (h *healthHandler) Trace(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")

	count := r.URL.Query().Get("count")

	c, err := strconv.Atoi(count)
	if err != nil {
		httputils.ErrResponse(w, errorutils.BadRequest("trace", "received bad count query param: %v", err))
		return
	}

	spans, err := h.tracer.Read(
		trace.ReadWithTrace(id),
		trace.ReadWithCount(c),
	)
	if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("trace", "failed to retrieve traces: %v", err))
		return
	}

	httputils.OkResponse(w, spans)
}

func NewHealthHandler(tracer trace.Trace) HealthHandler {
	return &healthHandler{tracer}
}
