package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
)

type StateHandler interface {
	HandlePost(w http.ResponseWriter, r *http.Request)
	HandleGet(w http.ResponseWriter, r *http.Request)
}

type stateHandler struct {
	action sidecar.Sidecar
}

func (h *stateHandler) HandlePost(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	defer r.Body.Close()

	if r.Body == nil {
		BadRequest(w, "expected a body as state")
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
	}

	w.WriteHeader(200)
	w.Write(nil)
}

func (h *stateHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	key := params["key"]

	recs, err := h.action.RetrieveStateFromStore(storeId, key)
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

func NewStateHandler(s sidecar.Sidecar) StateHandler {
	return &stateHandler{s}
}
