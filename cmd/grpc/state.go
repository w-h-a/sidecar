package grpc

import (
	"context"

	pb "github.com/w-h-a/pkg/proto/sidecar"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
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
}

func (h *stateHandler) Post(ctx context.Context, req *pb.PostStateRequest, rsp *pb.PostStateResponse) error {
	state := &sidecar.State{
		StoreId: req.StoreId,
		Records: DeserializeRecords(req.Records),
	}

	if err := h.service.SaveStateToStore(state); err != nil && err == sidecar.ErrComponentNotFound {
		return errorutils.NotFound("sidecar", "%v: %s", err, req.StoreId)
	} else if err != nil {
		return errorutils.InternalServerError("sidecar", "failed to save state to store %s: %v", req.StoreId, err)
	}

	return nil
}

func (h *stateHandler) List(ctx context.Context, req *pb.ListStateRequest, rsp *pb.ListStateResponse) error {
	recs, err := h.service.ListStateFromStore(req.StoreId)
	if err != nil && err == sidecar.ErrComponentNotFound {
		return errorutils.NotFound("sidecar", "%v: %s", err, req.StoreId)
	} else if err != nil {
		return errorutils.InternalServerError("sidecar", "failed to retrive state from store %s: %v", req.StoreId, err)
	}

	rsp.Records = SerializeRecords(recs)

	return nil
}

func (h *stateHandler) Get(ctx context.Context, req *pb.GetStateRequest, rsp *pb.GetStateResponse) error {
	recs, err := h.service.SingleStateFromStore(req.StoreId, req.Key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		return errorutils.NotFound("sidecar", "%v: %s", err, req.StoreId)
	} else if err != nil && err == store.ErrRecordNotFound {
		return errorutils.NotFound("sidecar", "there is no such record at store %s and key %s", req.StoreId, req.Key)
	} else if err != nil {
		return errorutils.InternalServerError("sidecar", "failed to retrieve state from store %s and key %s: %v", req.StoreId, req.Key, err)
	}

	rsp.Records = SerializeRecords(recs)

	return nil
}

func (h *stateHandler) Delete(ctx context.Context, req *pb.DeleteStateRequest, rsp *pb.DeleteStateResponse) error {
	if err := h.service.RemoveStateFromStore(req.StoreId, req.Key); err != nil && err == sidecar.ErrComponentNotFound {
		return errorutils.NotFound("sidecar", "%v: %s", err, req.StoreId)
	} else if err != nil {
		return errorutils.InternalServerError("sidecar", "failed to remove state from store %s and key %s: %v", req.StoreId, req.Key, err)
	}

	return nil
}

func NewStateHandler(s sidecar.Sidecar) StateHandler {
	return &State{&stateHandler{s}}
}
