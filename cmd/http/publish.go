package http

import (
	"encoding/json"
	"fmt"
	gohttp "net/http"

	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/telemetry/tracev2"
	"github.com/w-h-a/pkg/utils/errorutils"
	"github.com/w-h-a/pkg/utils/httputils"
	"github.com/w-h-a/pkg/utils/metadatautils"
)

type PublishHandler interface {
	Handle(w gohttp.ResponseWriter, r *gohttp.Request)
}

type publishHandler struct {
	service sidecar.Sidecar
	tracer  tracev2.Trace
}

func (h *publishHandler) Handle(w gohttp.ResponseWriter, r *gohttp.Request) {
	ctx := metadatautils.RequestToContext(r)

	newCtx, spanId := h.tracer.Start(ctx, "http.PublishHandler")
	defer h.tracer.Finish(spanId)

	defer r.Body.Close()

	if r.Body == nil {
		h.tracer.UpdateStatus(spanId, 1, "event is required")
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "event is required"))
		return
	}

	var event *sidecar.Event

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&event); err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to decode request: %v", err))
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "failed to decode request: %v", err))
		return
	}

	payload, _ := json.Marshal(event.Payload)

	h.tracer.AddMetadata(spanId, map[string]string{
		"eventName": event.EventName,
		"payload":   string(payload),
	})

	if len(event.EventName) == 0 {
		h.tracer.UpdateStatus(spanId, 1, "an event name as topic is required")
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "an event name as topic is required"))
		return
	}

	if err := h.service.WriteEventToBroker(newCtx, event); err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), event.EventName))
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), event.EventName))
		return
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to publish event: %v", err))
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to publish event: %v", err))
		return
	}

	h.tracer.UpdateStatus(spanId, 2, "success")

	httputils.OkResponse(w, map[string]interface{}{})
}

func NewPublishHandler(s sidecar.Sidecar, t tracev2.Trace) PublishHandler {
	return &publishHandler{s, t}
}
