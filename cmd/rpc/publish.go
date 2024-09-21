package rpc

import (
	"context"

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

	if len(req.Event.EventName) == 0 {
		return errorutils.BadRequest("sidecar", "an event name as topic is required")
	}

	event := &sidecar.Event{
		EventName: req.Event.EventName,
		Data:      req.Event.Data.Value,
	}

	if err := c.service.WriteEventToBroker(event); err != nil && err == sidecar.ErrComponentNotFound {
		return errorutils.NotFound("sidecar", "%v: %s", err, req.Event.EventName)
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
