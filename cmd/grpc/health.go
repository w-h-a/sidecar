package grpc

import (
	"context"

	pbHealth "github.com/w-h-a/pkg/proto/health"
	pbTrace "github.com/w-h-a/pkg/proto/trace"
	"github.com/w-h-a/pkg/telemetry/traceexporter"
	"github.com/w-h-a/pkg/utils/memoryutils"
)

type HealthHandler interface {
	Check(ctx context.Context, req *pbHealth.HealthRequest, rsp *pbHealth.HealthResponse) error
	Trace(ctx context.Context, req *pbTrace.TraceRequest, rsp *pbTrace.TraceResponse) error
}

type Health struct {
	HealthHandler
}

type healthHandler struct {
	buffer *memoryutils.Buffer
}

func (h *healthHandler) Check(ctx context.Context, req *pbHealth.HealthRequest, rsp *pbHealth.HealthResponse) error {
	rsp.Status = "ok"
	return nil
}

func (h *healthHandler) Trace(ctx context.Context, req *pbTrace.TraceRequest, rsp *pbTrace.TraceResponse) error {
	count := req.Count

	var entries []*memoryutils.Entry

	if count > 0 {
		entries = h.buffer.Get(int(req.Count))
	} else {
		entries = h.buffer.Get(h.buffer.Options().Size)
	}

	spans := []*traceexporter.SpanData{}

	for _, entry := range entries {
		span := entry.Value.(*traceexporter.SpanData)

		spans = append(spans, span)
	}

	for _, span := range spans {
		rsp.Spans = append(rsp.Spans, SerializeSpan(span))
	}

	return nil
}

func NewHealthHandler(b *memoryutils.Buffer) HealthHandler {
	return &Health{&healthHandler{b}}
}
