package http

import (
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

	newCtx := h.tracer.Start(ctx, "secretHandler")

	h.tracer.AddMetadata(map[string]string{
		"secretId": secretId,
		"key":      key,
	})

	secret, err := h.service.ReadFromSecretStore(newCtx, secretId, key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		// TODO: update span status
		h.tracer.Finish()
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), secretId))
		return
	} else if err != nil {
		// TODO: update span status
		h.tracer.Finish()
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve secret from store %s and key %s: %v", secretId, key, err))
		return
	}

	// TODO: update span status

	h.tracer.Finish()

	httputils.OkResponse(w, secret)
}

func NewSecretHandler(s sidecar.Sidecar, t tracev2.Trace) SecretHandler {
	return &secretHandler{s, t}
}
