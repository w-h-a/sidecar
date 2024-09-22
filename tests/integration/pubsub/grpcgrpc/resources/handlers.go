package resources

import (
	"context"
	"time"

	pbHealth "github.com/w-h-a/pkg/proto/health"
	pbSidecar "github.com/w-h-a/pkg/proto/sidecar"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type HealthHandler interface {
	Check(ctx context.Context, req *pbHealth.HealthRequest, rsp *pbHealth.HealthResponse) error
}

type Health struct {
	HealthHandler
}

type healthHandler struct {
}

func (h *healthHandler) Check(ctx context.Context, req *pbHealth.HealthRequest, rsp *pbHealth.HealthResponse) error {
	rsp.Status = "ok"
	return nil
}

func NewHealthHandler() HealthHandler {
	return &Health{&healthHandler{}}
}

type SubscribeHandler interface {
	A(ctx context.Context, req *pbSidecar.Event, rsp *pbSidecar.Event) error
	B(ctx context.Context, req *pbSidecar.Event, rsp *pbSidecar.Event) error
}

type Go struct {
	SubscribeHandler
}

type subscribeHandler struct {
	event chan *MethodEvent
}

func (h *subscribeHandler) A(ctx context.Context, req *pbSidecar.Event, rsp *pbSidecar.Event) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return errorutils.Timeout("grpc-subscriber", "timeout")
	case h.event <- &MethodEvent{Method: "go-a", Event: req}:
		return nil
	}
}

func (h *subscribeHandler) B(ctx context.Context, req *pbSidecar.Event, rsp *pbSidecar.Event) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		return errorutils.Timeout("grpc-subscriber", "timeout")
	case h.event <- &MethodEvent{Method: "go-b", Event: req}:
		return nil
	}
}

func NewSubscribeHandler(event chan *MethodEvent) SubscribeHandler {
	return &Go{&subscribeHandler{event}}
}
