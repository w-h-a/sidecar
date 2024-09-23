package http

import (
	"encoding/json"
	"net/http"

	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type PublishHandler interface {
	Handle(w http.ResponseWriter, r *http.Request)
}

type publishHandler struct {
	service sidecar.Sidecar
}

func (h *publishHandler) Handle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Body == nil {
		ErrResponse(w, errorutils.BadRequest("sidecar", "event is required"))
		return
	}

	var event *sidecar.Event

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&event); err != nil {
		ErrResponse(w, errorutils.BadRequest("sidecar", "failed to decode request: %v", err))
		return
	}

	if len(event.EventName) == 0 {
		ErrResponse(w, errorutils.BadRequest("sidecar", "an event name as topic is required"))
		return
	}

	if err := h.service.WriteEventToBroker(event); err != nil && err == sidecar.ErrComponentNotFound {
		ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), event.EventName))
		return
	} else if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to publish event: %v", err))
		return
	}

	w.WriteHeader(200)
	w.Write(nil)
}

func NewPublishHandler(s sidecar.Sidecar) PublishHandler {
	return &publishHandler{s}
}
