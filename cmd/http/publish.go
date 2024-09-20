package http

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/w-h-a/pkg/sidecar"
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
		BadRequest(w, "expected a body as event")
		return
	}

	var event *sidecar.Event

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&event); err != nil {
		BadRequest(w, "failed to decode request: "+err.Error())
		return
	}

	if len(event.To) == 0 {
		BadRequest(w, "an address/topic to send to is required")
		return
	}

	if err := h.service.WriteEventToBroker(event); err != nil && err == sidecar.ErrComponentNotFound {
		w.WriteHeader(404)
		w.Write([]byte(fmt.Sprintf("%s: %#+v", err.Error(), event.To)))
		return
	} else if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(200)
	w.Write(nil)
}

func NewPublishHandler(s sidecar.Sidecar) PublishHandler {
	return &publishHandler{s}
}
