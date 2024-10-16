package http

import (
	"encoding/json"
	"fmt"
	gohttp "net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/telemetry/tracev2"
	"github.com/w-h-a/pkg/utils/errorutils"
	"github.com/w-h-a/pkg/utils/httputils"
	"github.com/w-h-a/pkg/utils/metadatautils"
)

type StateHandler interface {
	HandlePost(w gohttp.ResponseWriter, r *gohttp.Request)
	HandleList(w gohttp.ResponseWriter, r *gohttp.Request)
	HandleGet(w gohttp.ResponseWriter, r *gohttp.Request)
	HandleDelete(w gohttp.ResponseWriter, r *gohttp.Request)
}

type stateHandler struct {
	service sidecar.Sidecar
	tracer  tracev2.Trace
}

func (h *stateHandler) HandlePost(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	ctx := metadatautils.RequestToContext(r)

	newCtx, spanId := h.tracer.Start(ctx, "http.PostStateHandler")
	defer h.tracer.Finish(spanId)

	defer r.Body.Close()

	if r.Body == nil {
		h.tracer.UpdateStatus(spanId, 1, "expected a body as array of records")
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "expected a body as array of records"))
		return
	}

	var records []sidecar.Record

	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&records); err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to decode request: %v", err))
		httputils.ErrResponse(w, errorutils.BadRequest("sidecar", "failed to decode request: %v", err))
		return
	}

	bytes, _ := json.Marshal(records)

	h.tracer.AddMetadata(spanId, map[string]string{
		"storeId": storeId,
		"records": string(bytes),
	})

	state := &sidecar.State{
		StoreId: storeId,
		Records: records,
	}

	if err := h.service.SaveStateToStore(newCtx, state); err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), storeId))
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to save state to store %s: %v", storeId, err))
		httputils.ErrResponse(w, errorutils.InternalServerError("failed to save state to store %s: %v", storeId, err))
		return
	}

	h.tracer.UpdateStatus(spanId, 2, "success")

	httputils.OkResponse(w, map[string]interface{}{})
}

func (h *stateHandler) HandleList(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	ctx := metadatautils.RequestToContext(r)

	newCtx, spanId := h.tracer.Start(ctx, "http.ListStateHandler")
	defer h.tracer.Finish(spanId)

	recs, err := h.service.ListStateFromStore(newCtx, storeId)
	if err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), storeId))
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to retrieve state from store %s: %v", storeId, err))
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve state from store %s: %v", storeId, err))
		return
	}

	if len(recs) == 0 {
		h.tracer.UpdateStatus(spanId, 2, "success")
		httputils.OkResponse(w, []sidecar.Record{})
		return
	}

	sidecarRecords, err := SerializeRecords(recs)
	if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to serialize records: %v", err))
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to serialize records: %v", err))
		return
	}

	h.tracer.UpdateStatus(spanId, 2, "success")

	httputils.OkResponse(w, sidecarRecords)
}

func (h *stateHandler) HandleGet(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	key := params["key"]

	ctx := metadatautils.RequestToContext(r)

	newCtx, spanId := h.tracer.Start(ctx, "http.GetStateHandler")
	defer h.tracer.Finish(spanId)

	h.tracer.AddMetadata(spanId, map[string]string{
		"storeId": storeId,
		"key":     key,
	})

	recs, err := h.service.SingleStateFromStore(newCtx, storeId, key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), storeId))
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil && err == store.ErrRecordNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("there is no such record at store %s and key %s: %v", storeId, key, err))
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "there is no such record at store %s and key %s: %v", storeId, key, err))
		return
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to retrieve state from store %s and key %s: %v", storeId, key, err))
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve state from store %s and key %s: %v", storeId, key, err))
		return
	}

	if len(recs) == 0 {
		h.tracer.UpdateStatus(spanId, 2, "success")
		httputils.OkResponse(w, []sidecar.Record{})
		return
	}

	sidecarRecords, err := SerializeRecords(recs)
	if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to serialize records: %v", err))
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to serialize records: %v", err))
		return
	}

	h.tracer.UpdateStatus(spanId, 2, "success")

	httputils.OkResponse(w, sidecarRecords)
}

func (h *stateHandler) HandleDelete(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	storeId := params["storeId"]

	key := params["key"]

	ctx := metadatautils.RequestToContext(r)

	newCtx, spanId := h.tracer.Start(ctx, "http.DeleteStateHandler")
	defer h.tracer.Finish(spanId)

	h.tracer.AddMetadata(spanId, map[string]string{
		"storeId": storeId,
		"key":     key,
	})

	if err := h.service.RemoveStateFromStore(newCtx, storeId, key); err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), storeId))
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), storeId))
		return
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to remove state from store %s and key %s: %v", storeId, key, err))
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to remove state from store %s and key %s: %v", storeId, key, err))
		return
	}

	h.tracer.UpdateStatus(spanId, 2, "success")

	httputils.OkResponse(w, map[string]interface{}{})
}

func NewStateHandler(s sidecar.Sidecar, t tracev2.Trace) StateHandler {
	return &stateHandler{s, t}
}
