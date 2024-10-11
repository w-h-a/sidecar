package grpc

import (
	"context"

	pbHealth "github.com/w-h-a/pkg/proto/health"
	pbTrace "github.com/w-h-a/pkg/proto/trace"
	"github.com/w-h-a/pkg/telemetry/trace"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type HealthHandler interface {
	Check(ctx context.Context, req *pbHealth.HealthRequest, rsp *pbHealth.HealthResponse) error
	Trace(ctx context.Context, req *pbTrace.TraceRequest, rsp *pbTrace.TraceResponse) error
}

type Health struct {
	HealthHandler
}

type healthHandler struct {
	tracer trace.Trace
}

func (h *healthHandler) Check(ctx context.Context, req *pbHealth.HealthRequest, rsp *pbHealth.HealthResponse) error {
	rsp.Status = "ok"
	return nil
}

func (h *healthHandler) Trace(ctx context.Context, req *pbTrace.TraceRequest, rsp *pbTrace.TraceResponse) error {
	spans, err := h.tracer.Read(
		trace.ReadWithTrace(req.Id),
		trace.ReadWithCount(int(req.Count)),
	)
	if err != nil {
		return errorutils.InternalServerError("trace", "failed to retrieve traces: %v", err)
	}

	for _, span := range spans {
		rsp.Spans = append(rsp.Spans, SerializeSpan(span))
	}

	return nil
}

func NewHealthHandler(t trace.Trace) HealthHandler {
	return &Health{&healthHandler{t}}
}
