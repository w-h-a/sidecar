package grpc

import (
	"context"
	"fmt"

	pb "github.com/w-h-a/pkg/proto/sidecar"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/telemetry/tracev2"
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
	tracer  tracev2.Trace
}

func (h *secretHandler) Get(ctx context.Context, req *pb.GetSecretRequest, rsp *pb.GetSecretResponse) error {
	newCtx, spanId := h.tracer.Start(ctx, "grpc.SecretHandler")
	defer h.tracer.Finish(spanId)

	h.tracer.AddMetadata(spanId, map[string]string{
		"secretId": req.SecretId,
		"key":      req.Key,
	})

	secret, err := h.service.ReadFromSecretStore(newCtx, req.SecretId, req.Key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("%s: %s", err.Error(), req.SecretId))
		return errorutils.NotFound("sidecar", "%v: %s", err, req.SecretId)
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 1, fmt.Sprintf("failed to retrieve secret from store %s and key %s: %v", req.SecretId, req.Key, err))
		return errorutils.InternalServerError("sidecar", "failed to retrieve secret from store %s and key %s: %v", req.SecretId, req.Key, err)
	}

	rsp.Secret = SerializeSecret(secret)

	h.tracer.UpdateStatus(spanId, 2, "success")

	return nil
}

func NewSecretHandler(s sidecar.Sidecar, t tracev2.Trace) SecretHandler {
	return &Secret{&secretHandler{s, t}}
}
