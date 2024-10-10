package http

import (
	"encoding/json"
	gohttp "net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/utils/errorutils"
	"github.com/w-h-a/pkg/utils/httputils"
)

type StateHandler interface {
	HandlePost(w gohttp.ResponseWriter, r *gohttp.Request)
	HandleList(w gohttp.ResponseWriter, r *gohttp.Request)
	HandleGet(w gohttp.ResponseWriter, r *gohttp.Request)
	HandleDelete(w gohttp.ResponseWriter, r *gohttp.Request)
}

type stateHandler struct {
	service sidecar.Sidecar
}

func (h *stateHandler) HandlePost(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	defer r.Body.Close()

	if r.Body == nil {
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "expected a body as array of records"))
		return
	}

	var records []sidecar.Record

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&records); err != nil {
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "failed to decode request: "+err.Error()))
		return
	}

	state := &sidecar.State{
		StoreId: storeId,
		Records: records,
	}

	if err := h.service.SaveStateToStore(state); err != nil && err == sidecar.ErrComponentNotFound {
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("failed to save state to store %s: %v", storeId, err))
		return
	}

	httputils.OkResponse(w, map[string]interface{}{})
}

func (h *stateHandler) HandleList(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	recs, err := h.service.ListStateFromStore(storeId)
	if err != nil && err == sidecar.ErrComponentNotFound {
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve state from store %s: %v", storeId, err))
		return
	}

	if len(recs) == 0 {
		httputils.OkResponse(w, []sidecar.Record{})
		return
	}

	sidecarRecords, err := SerializeRecords(recs)
	if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to serialize records: %v", err))
		return
	}

	httputils.OkResponse(w, sidecarRecords)
}

func (h *stateHandler) HandleGet(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	key := params["key"]

	recs, err := h.service.SingleStateFromStore(storeId, key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil && err == store.ErrRecordNotFound {
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "there is no such record at store %s and key %s: %v", storeId, key, err))
		return
	} else if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve state from store %s and key %s: %v", storeId, key, err))
		return
	}

	if len(recs) == 0 {
		httputils.OkResponse(w, []sidecar.Record{})
		return
	}

	sidecarRecords, err := SerializeRecords(recs)
	if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to serialize records: %v", err))
		return
	}

	httputils.OkResponse(w, sidecarRecords)
}

func (h *stateHandler) HandleDelete(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	key := params["key"]

	if err := h.service.RemoveStateFromStore(storeId, key); err != nil && err == sidecar.ErrComponentNotFound {
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to remove state from store %s and key %s: %v", storeId, key, err))
		return
	}

	httputils.OkResponse(w, map[string]interface{}{})
}

func NewStateHandler(s sidecar.Sidecar) StateHandler {
	return &stateHandler{s}
}
