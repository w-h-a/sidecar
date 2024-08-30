package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
)

type StateHandler interface {
	HandlePost(w http.ResponseWriter, r *http.Request)
	HandleList(w http.ResponseWriter, r *http.Request)
	HandleGet(w http.ResponseWriter, r *http.Request)
	HandleDelete(w http.ResponseWriter, r *http.Request)
}

type stateHandler struct {
	action sidecar.Sidecar
}

func (h *stateHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	defer r.Body.Close()

	if r.Body == nil {
		BadRequest(w, "expected a body as array of records")
		return
	}

	var records []sidecar.Record

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&records); err != nil {
		BadRequest(w, "failed to decode request: "+err.Error())
		return
	}

	state := &sidecar.State{
		StoreId: storeId,
		Records: records,
	}

	if err := h.action.SaveStateToStore(state); err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(200)
	w.Write(nil)
}

func (h *stateHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	recs, err := h.action.ListStateFromStore(storeId)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	if len(recs) == 0 {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
		return
	}

	sidecarRecords, err := SerializeRecords(recs)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	bs, err := json.Marshal(sidecarRecords)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
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

	recs, err := h.action.SingleStateFromStore(storeId, key)
	if err != nil && err == store.ErrRecordNotFound {
		w.WriteHeader(404)
		// TODO: json error responses
		w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, err.Error())))
		return
	} else if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	if len(recs) == 0 {
		w.WriteHeader(200)
		w.Write([]byte(`[]`))
		return
	}

	sidecarRecords, err := SerializeRecords(recs)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	bs, err := json.Marshal(sidecarRecords)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
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

	if err := h.action.RemoveStateFromStore(storeId, key); err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(200)
	w.Write(nil)
}

func NewStateHandler(s sidecar.Sidecar) StateHandler {
	return &stateHandler{s}
}
