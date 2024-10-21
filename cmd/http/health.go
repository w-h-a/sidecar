package http

import (
	"net/http"
	"strconv"

	"github.com/w-h-a/pkg/telemetry/traceexporter"
	"github.com/w-h-a/pkg/utils/errorutils"
	"github.com/w-h-a/pkg/utils/httputils"
	"github.com/w-h-a/pkg/utils/memoryutils"
)

type HealthHandler interface {
	Check(w http.ResponseWriter, r *http.Request)
	Trace(w http.ResponseWriter, r *http.Request)
}

type healthHandler struct {
	buffer *memoryutils.Buffer
}

func (h *healthHandler) Check(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	w.Write([]byte("ok"))
}

func (h *healthHandler) Trace(w http.ResponseWriter, r *http.Request) {
	c := r.URL.Query().Get("count")

	var count int
	var err error

	if len(c) > 0 {
		count, err = strconv.Atoi(c)
		if err != nil {
			httputils.ErrResponse(w, errorutils.BadRequest("trace", "received bad count query param: %v", err))
			return
		}
	}

	var entries []*memoryutils.Entry

	if count > 0 {
		entries = h.buffer.Get(count)
	} else {
		entries = h.buffer.Get(h.buffer.Options().Size)
	}

	spans := []*traceexporter.SpanData{}

	for _, entry := range entries {
		span := entry.Value.(*traceexporter.SpanData)

		spans = append(spans, span)
	}

	httputils.OkResponse(w, spans)
}

func NewHealthHandler(b *memoryutils.Buffer) HealthHandler {
	return &healthHandler{b}
}
