package grpc

import (
	"context"

	pb "github.com/w-h-a/pkg/proto/health"
)

type HealthHandler interface {
	Check(ctx context.Context, req *pb.HealthRequest, rsp *pb.HealthResponse) error
}

type Health struct {
	HealthHandler
}

type healthHandler struct {
}

func (h *healthHandler) Check(ctx context.Context, req *pb.HealthRequest, rsp *pb.HealthResponse) error {
	rsp.Status = "ok"
	return nil
}

func NewHealthHandler() HealthHandler {
	return &Health{&healthHandler{}}
}
