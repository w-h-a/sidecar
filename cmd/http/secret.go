package http

import (
	"fmt"
	gohttp "net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/telemetry/tracev2"
	"github.com/w-h-a/pkg/utils/errorutils"
	"github.com/w-h-a/pkg/utils/httputils"
	"github.com/w-h-a/pkg/utils/metadatautils"
)

type SecretHandler interface {
	HandleGet(w gohttp.ResponseWriter, r *gohttp.Request)
}

type secretHandler struct {
	service sidecar.Sidecar
	tracer  tracev2.Trace
}

func (h *secretHandler) HandleGet(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	secretId := params["secretId"]

	key := params["key"]

	ctx := metadatautils.RequestToContext(r)

	newCtx, spanId := h.tracer.Start(ctx, "secretHandler")

	h.tracer.AddMetadata(spanId, map[string]string{
		"secretId": secretId,
		"key":      key,
	})

	secret, err := h.service.ReadFromSecretStore(newCtx, secretId, key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		h.tracer.UpdateStatus(spanId, 404, fmt.Sprintf("%s: %s", err.Error(), secretId))
		h.tracer.Finish(spanId)
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), secretId))
		return
	} else if err != nil {
		h.tracer.UpdateStatus(spanId, 500, fmt.Sprintf("failed to retrieve secret from store %s and key %s: %v", secretId, key, err))
		h.tracer.Finish(spanId)
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve secret from store %s and key %s: %v", secretId, key, err))
		return
	}

	h.tracer.UpdateStatus(spanId, 200, "successfully retrieved secret")

	h.tracer.Finish(spanId)

	httputils.OkResponse(w, secret)
}

func NewSecretHandler(s sidecar.Sidecar, t tracev2.Trace) SecretHandler {
	return &secretHandler{s, t}
}
