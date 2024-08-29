package controllers

import (
	"context"
	"time"

	pb "github.com/w-h-a/pkg/proto/action"
	"github.com/w-h-a/pkg/server"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type PublishController interface {
	Publish(ctx context.Context, req *pb.PublishRequest, rsp *pb.PublishResponse) error
}

type publishController struct {
	action sidecar.Sidecar
}

func (c *publishController) Publish(ctx context.Context, req *pb.PublishRequest, rsp *pb.PublishResponse) error {
	if req.Event == nil {
		return errorutils.BadRequest("action", "event is required")
	}

	event := &sidecar.Event{
		EventName:  req.Event.EventName,
		To:         req.Event.To,
		Concurrent: req.Event.Concurrent,
		Data:       req.Event.Data.Value,
		CreatedAt:  time.Now(),
	}

	if err := c.action.WriteEventToBroker(event); err != nil {
		return errorutils.InternalServerError("action", "failed to publish event: %v", err)
	}

	return nil
}

func NewPublishController(s sidecar.Sidecar) PublishController {
	return &publishController{s}
}

type Publish struct {
	PublishController
}

func RegisterPublishController(s server.Server, controller PublishController, opts ...server.ControllerOption) error {
	return s.RegisterController(
		s.NewController(
			&Publish{controller},
			opts...,
		),
	)
}
