package http

import (
	"encoding/json"
	gohttp "net/http"

	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/utils/errorutils"
	"github.com/w-h-a/pkg/utils/httputils"
)

type PublishHandler interface {
	Handle(w gohttp.ResponseWriter, r *gohttp.Request)
}

type publishHandler struct {
	service sidecar.Sidecar
}

func (h *publishHandler) Handle(w gohttp.ResponseWriter, r *gohttp.Request) {
	defer r.Body.Close()

	if r.Body == nil {
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "event is required"))
		return
	}

	var event *sidecar.Event

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&event); err != nil {
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "failed to decode request: %v", err))
		return
	}

	if len(event.EventName) == 0 {
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "an event name as topic is required"))
		return
	}

	if err := h.service.WriteEventToBroker(event); err != nil && err == sidecar.ErrComponentNotFound {
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), event.EventName))
		return
	} else if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to publish event: %v", err))
		return
	}

	httputils.OkResponse(w, map[string]interface{}{})
}

func NewPublishHandler(s sidecar.Sidecar) PublishHandler {
	return &publishHandler{s}
}
