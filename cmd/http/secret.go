package http

import (
	gohttp "net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/telemetry/trace"
	"github.com/w-h-a/pkg/utils/errorutils"
	"github.com/w-h-a/pkg/utils/httputils"
	"github.com/w-h-a/pkg/utils/metadatautils"
)

type SecretHandler interface {
	HandleGet(w gohttp.ResponseWriter, r *gohttp.Request)
}

type secretHandler struct {
	service sidecar.Sidecar
}

func (h *secretHandler) HandleGet(w gohttp.ResponseWriter, r *gohttp.Request) {
	params := mux.Vars(r)

	secretId := params["secretId"]

	key := params["key"]

	ctx := metadatautils.RequestToContext(r)

	tracer := trace.GetTracer()

	_, span, err := tracer.Start(
		ctx,
		"secretHandler",
		map[string]string{
			secretId: secretId,
			key:      key,
		},
	)
	if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to generate span: %v", err))
		return
	}

	// TODO: pass down context!
	secret, err := h.service.ReadFromSecretStore(secretId, key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		span.Metadata["error"] = err.Error()
		tracer.Finish(span)
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), secretId))
		return
	} else if err != nil {
		span.Metadata["error"] = err.Error()
		tracer.Finish(span)
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve secret from store %s and key %s: %v", secretId, key, err))
		return
	}

	tracer.Finish(span)

	httputils.OkResponse(w, secret)
}

func NewSecretHandler(s sidecar.Sidecar) SecretHandler {
	return &secretHandler{s}
}
