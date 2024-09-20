package rpc

import (
	"context"
	"time"

	pb "github.com/w-h-a/pkg/proto/sidecar"
	"github.com/w-h-a/pkg/server"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type PublishHandler interface {
	Publish(ctx context.Context, req *pb.PublishRequest, rsp *pb.PublishResponse) error
}

type publishHandler struct {
	service sidecar.Sidecar
}

func (c *publishHandler) Publish(ctx context.Context, req *pb.PublishRequest, rsp *pb.PublishResponse) error {
	if req.Event == nil {
		return errorutils.BadRequest("sidecar", "event is required")
	}

	event := &sidecar.Event{
		EventName:  req.Event.EventName,
		To:         req.Event.To,
		Concurrent: req.Event.Concurrent,
		Data:       req.Event.Data.Value,
		CreatedAt:  time.Now(),
	}

	if err := c.service.WriteEventToBroker(event); err != nil && err == sidecar.ErrComponentNotFound {
		return errorutils.NotFound("sidecar", "%v: %#+v", err, req.Event.To)
	} else if err != nil {
		return errorutils.InternalServerError("sidecar", "failed to publish event: %v", err)
	}

	return nil
}

func NewPublishHandler(s sidecar.Sidecar) PublishHandler {
	return &publishHandler{s}
}

type Publish struct {
	PublishHandler
}

func RegisterPublishHandler(s server.Server, handler PublishHandler, opts ...server.HandlerOption) error {
	return s.Handle(
		s.NewHandler(
			&Publish{handler},
			opts...,
		),
	)
}
