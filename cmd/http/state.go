package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type StateHandler interface {
	HandlePost(w http.ResponseWriter, r *http.Request)
	HandleList(w http.ResponseWriter, r *http.Request)
	HandleGet(w http.ResponseWriter, r *http.Request)
	HandleDelete(w http.ResponseWriter, r *http.Request)
}

type stateHandler struct {
	service sidecar.Sidecar
}

func (h *stateHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	defer r.Body.Close()

	if r.Body == nil {
		ErrResponse(w, errorutils.BadRequest("sidecar", "expected a body as array of records"))
		return
	}

	var records []sidecar.Record

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&records); err != nil {
		ErrResponse(w, errorutils.BadRequest("sidecar", "failed to decode request: "+err.Error()))
		return
	}

	state := &sidecar.State{
		StoreId: storeId,
		Records: records,
	}

	if err := h.service.SaveStateToStore(state); err != nil && err == sidecar.ErrComponentNotFound {
		ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil {
		ErrResponse(w, errorutils.InternalServerError("failed to save state to store %s: %v", storeId, err))
		return
	}

	w.WriteHeader(200)
	w.Write(nil)
}

func (h *stateHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	recs, err := h.service.ListStateFromStore(storeId)
	if err != nil && err == sidecar.ErrComponentNotFound {
		ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve state from store %s: %v", storeId, err))
		return
	}

	if len(recs) == 0 {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
		return
	}

	sidecarRecords, err := SerializeRecords(recs)
	if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to serialize records: %v", err))
		return
	}

	bs, err := json.Marshal(sidecarRecords)
	if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to marshal records: %v", err))
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	w.Write(bs)
}

func (h *stateHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	key := params["key"]

	recs, err := h.service.SingleStateFromStore(storeId, key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil && err == store.ErrRecordNotFound {
		ErrResponse(w, errorutils.NotFound("sidecar", "there is no such record at store %s and key %s: %v", storeId, key, err))
		return
	} else if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve state from store %s and key %s: %v", storeId, key, err))
		return
	}

	if len(recs) == 0 {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
		return
	}

	sidecarRecords, err := SerializeRecords(recs)
	if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to serialize records: %v", err))
		return
	}

	bs, err := json.Marshal(sidecarRecords)
	if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to marshal records: %v", err))
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	w.Write(bs)
}

func (h *stateHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	key := params["key"]

	if err := h.service.RemoveStateFromStore(storeId, key); err != nil && err == sidecar.ErrComponentNotFound {
		ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to remove state from store %s and key %s: %v", storeId, key, err))
		return
	}

	w.WriteHeader(200)
	w.Write(nil)
}

func NewStateHandler(s sidecar.Sidecar) StateHandler {
	return &stateHandler{s}
}
