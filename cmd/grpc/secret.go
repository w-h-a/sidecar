package grpc

import (
	"context"

	pb "github.com/w-h-a/pkg/proto/sidecar"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type SecretHandler interface {
	Get(ctx context.Context, req *pb.GetSecretRequest, rsp *pb.GetSecretResponse) error
}

type Secret struct {
	SecretHandler
}

type secretHandler struct {
	service sidecar.Sidecar
}

func (h *secretHandler) Get(ctx context.Context, req *pb.GetSecretRequest, rsp *pb.GetSecretResponse) error {
	secret, err := h.service.ReadFromSecretStore(ctx, req.SecretId, req.Key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		return errorutils.NotFound("sidecar", "%v: %s", err, req.SecretId)
	} else if err != nil {
		return errorutils.InternalServerError("sidecar", "failed to retrieve secret from store %s and key %s: %v", req.SecretId, req.Key, err)
	}

	rsp.Secret = SerializeSecret(secret)

	return nil
}

func NewSecretHandler(s sidecar.Sidecar) SecretHandler {
	return &Secret{&secretHandler{s}}
}
