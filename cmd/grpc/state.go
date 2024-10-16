package grpc

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/w-h-a/pkg/proto/sidecar"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/telemetry/tracev2"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type StateHandler interface {
	Post(ctx context.Context, req *pb.PostStateRequest, rsp *pb.PostStateResponse) error
	List(ctx context.Context, req *pb.ListStateRequest, rsp *pb.ListStateResponse) error
	Get(ctx context.Context, req *pb.GetStateRequest, rsp *pb.GetStateResponse) error
	Delete(ctx context.Context, req *pb.DeleteStateRequest, rsp *pb.DeleteStateResponse) error
}

type State struct {
	StateHandler
}

type stateHandler struct {
	service sidecar.Sidecar
	tracer  tracev2.Trace
}

func (h *stateHandler) Post(ctx context.Context, req *pb.PostStateRequest, rsp *pb.PostStateResponse) error {
	newCtx, spanId := h.tracer.Start(ctx, "grpc.PostStateHandler")
	defer h.tracer.Finish(spanId)

	records, _ := json.Marshal(req.Records)

	h.tracer.AddMetadata(spanId, map[string]string{
		"storeId": req.StoreId,
		"records": string(records),
	})

	state := &sidecar.State{
		StoreId: req.StoreId,
		Records: DeserializeRecords(req.Records),
	}

	if err := h.service.SaveStateToStore(newCtx, state); err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), req.StoreId))
		return errorutils.NotFound("sidecar", "%v: %s", err, req.StoreId)
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to save state to store %s: %v", req.StoreId, err))
		return errorutils.InternalServerError("sidecar", "failed to save state to store %s: %v", req.StoreId, err)
	}

	h.tracer.UpdateStatus(spanId, 2, "success")

	return nil
}

func (h *stateHandler) List(ctx context.Context, req *pb.ListStateRequest, rsp *pb.ListStateResponse) error {
	newCtx, spanId := h.tracer.Start(ctx, "grpc.ListStateHandler")
	defer h.tracer.Finish(spanId)

	h.tracer.AddMetadata(spanId, map[string]string{
		"storeId": req.StoreId,
	})

	recs, err := h.service.ListStateFromStore(newCtx, req.StoreId)
	if err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), req.StoreId))
		return errorutils.NotFound("sidecar", "%v: %s", err, req.StoreId)
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to retrieve state from store %s: %v", req.StoreId, err))
		return errorutils.InternalServerError("sidecar", "failed to retrieve state from store %s: %v", req.StoreId, err)
	}

	rsp.Records = SerializeRecords(recs)

	h.tracer.UpdateStatus(spanId, 2, "success")

	return nil
}

func (h *stateHandler) Get(ctx context.Context, req *pb.GetStateRequest, rsp *pb.GetStateResponse) error {
	newCtx, spanId := h.tracer.Start(ctx, "grpc.GetStateHandler")
	defer h.tracer.Finish(spanId)

	h.tracer.AddMetadata(spanId, map[string]string{
		"storeId": req.StoreId,
		"key":     req.Key,
	})

	recs, err := h.service.SingleStateFromStore(newCtx, req.StoreId, req.Key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), req.StoreId))
		return errorutils.NotFound("sidecar", "%v: %s", err, req.StoreId)
	} else if err != nil && err == store.ErrRecordNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("there is no such record at store %s and key %s", req.StoreId, req.Key))
		return errorutils.NotFound("sidecar", "there is no such record at store %s and key %s", req.StoreId, req.Key)
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to retrieve state from store %s and key %s: %v", req.StoreId, req.Key, err))
		return errorutils.InternalServerError("sidecar", "failed to retrieve state from store %s and key %s: %v", req.StoreId, req.Key, err)
	}

	rsp.Records = SerializeRecords(recs)

	h.tracer.UpdateStatus(spanId, 2, "success")

	return nil
}

func (h *stateHandler) Delete(ctx context.Context, req *pb.DeleteStateRequest, rsp *pb.DeleteStateResponse) error {
	newCtx, spanId := h.tracer.Start(ctx, "grpc.DeleteStateHandler")
	defer h.tracer.Finish(spanId)

	h.tracer.AddMetadata(spanId, map[string]string{
		"storeId": req.StoreId,
		"key":     req.Key,
	})

	if err := h.service.RemoveStateFromStore(newCtx, req.StoreId, req.Key); err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), req.StoreId))
		return errorutils.NotFound("sidecar", "%v: %s", err, req.StoreId)
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to remove state from store %s and key %s: %v", req.StoreId, req.Key, err))
		return errorutils.InternalServerError("sidecar", "failed to remove state from store %s and key %s: %v", req.StoreId, req.Key, err)
	}

	h.tracer.UpdateStatus(spanId, 2, "success")

	return nil
}

func NewStateHandler(s sidecar.Sidecar, t tracev2.Trace) StateHandler {
	return &State{&stateHandler{s, t}}
}
