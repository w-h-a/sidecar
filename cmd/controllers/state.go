package controllers

import (
	"context"

	pb "github.com/w-h-a/pkg/proto/action"
	"github.com/w-h-a/pkg/server"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/store"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type StateController interface {
	Post(ctx context.Context, req *pb.PostStateRequest, rsp *pb.PostStateResponse) error
	List(ctx context.Context, req *pb.ListStateRequest, rsp *pb.ListStateResponse) error
	Get(ctx context.Context, req *pb.GetStateRequest, rsp *pb.GetStateResponse) error
	Delete(ctx context.Context, req *pb.DeleteStateRequest, rsp *pb.DeleteStateResponse) error
}

type stateController struct {
	action sidecar.Sidecar
}

func (c *stateController) Post(ctx context.Context, req *pb.PostStateRequest, rsp *pb.PostStateResponse) error {
	state := &sidecar.State{
		StoreId: req.StoreId,
		Records: DeserializeRecords(req.Records),
	}

	if err := c.action.SaveStateToStore(state); err != nil {
		return errorutils.InternalServerError("action", "failed to save state to store %s: %v", req.StoreId, err)
	}

	return nil
}

func (c *stateController) List(ctx context.Context, req *pb.ListStateRequest, rsp *pb.ListStateResponse) error {
	recs, err := c.action.ListStateFromStore(req.StoreId)
	if err != nil {
		return errorutils.InternalServerError("action", "failed to retrive state from store %s: %v", req.StoreId, err)
	}

	rsp.Records = SerializeRecords(recs)

	return nil
}

func (c *stateController) Get(ctx context.Context, req *pb.GetStateRequest, rsp *pb.GetStateResponse) error {
	recs, err := c.action.SingleStateFromStore(req.StoreId, req.Key)
	if err != nil && err == store.ErrRecordNotFound {
		return errorutils.NotFound("action", "there is no such record at store %s and key %s", req.StoreId, req.Key)
	} else if err != nil {
		return errorutils.InternalServerError("action", "failed to retrieve state from store %s and key %s: %v", req.StoreId, req.Key, err)
	}

	rsp.Records = SerializeRecords(recs)

	return nil
}

func (c *stateController) Delete(ctx context.Context, req *pb.DeleteStateRequest, rsp *pb.DeleteStateResponse) error {
	if err := c.action.RemoveStateFromStore(req.StoreId, req.Key); err != nil {
		return errorutils.InternalServerError("action", "failed to remove state from store %s and key %s: %v", req.StoreId, req.Key, err)
	}

	return nil
}

func NewStateController(s sidecar.Sidecar) StateController {
	return &stateController{s}
}

type State struct {
	StateController
}

func RegisterStateController(s server.Server, controller StateController, opts ...server.ControllerOption) error {
	return s.RegisterController(
		s.NewController(
			&State{controller},
			opts...,
		),
	)
}
