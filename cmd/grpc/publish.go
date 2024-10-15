package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/w-h-a/pkg/proto/sidecar"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/telemetry/tracev2"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type PublishHandler interface {
	Publish(ctx context.Context, req *pb.PublishRequest, rsp *pb.PublishResponse) error
}

type Publish struct {
	PublishHandler
}

type publishHandler struct {
	service sidecar.Sidecar
	tracer  tracev2.Trace
}

func (h *publishHandler) Publish(ctx context.Context, req *pb.PublishRequest, rsp *pb.PublishResponse) error {
	newCtx, spanId := h.tracer.Start(ctx, "grpc.PublishHandler")
	defer h.tracer.Finish(spanId)

	if req.Event == nil {
		h.tracer.UpdateStatus(spanId, 1, "event is required")
		return errorutils.BadRequest("sidecar", "event is required")
	}

	payload, _ := json.Marshal(req.Event.Payload)

	h.tracer.AddMetadata(spanId, map[string]string{
		"eventName": req.Event.EventName,
		"payload":   string(payload),
	})

	if len(req.Event.EventName) == 0 {
		h.tracer.UpdateStatus(spanId, 1, "an event name as topic is required")
		return errorutils.BadRequest("sidecar", "an event name as topic is required")
	}

	event := &sidecar.Event{
		EventName: req.Event.EventName,
		Payload: sidecar.Payload{
			Metadata: req.Event.Payload.Metadata,
			Data:     req.Event.Payload.Data,
		},
	}

	if err := h.service.WriteEventToBroker(newCtx, event); err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), req.Event.EventName))
		return errorutils.NotFound("sidecar", "%v: %s", err, req.Event.EventName)
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to publish event: %v", err))
		return errorutils.InternalServerError("sidecar", "failed to publish event: %v", err)
	}

	h.tracer.UpdateStatus(spanId, 2, "success")

	return nil
}

func NewPublishHandler(s sidecar.Sidecar, t tracev2.Trace) PublishHandler {
	return &Publish{&publishHandler{s, t}}
}
