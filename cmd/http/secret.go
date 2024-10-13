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

	ctx, span, err := h.tracer.Start(metadatautils.RequestToContext(r), "secretHandler")
	if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to generate span: %v", err))
		return
	}

	h.tracer.AddMetadata(span, map[string]string{
		"secretId": secretId,
		"key":      key,
	})

	newCtx, err := tracev2.ContextWithSpanData(ctx, span.SpanData())
	if err != nil {
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to pass span: %v", err))
		return
	}

	secret, err := h.service.ReadFromSecretStore(newCtx, secretId, key)
	if err != nil && err == tracev2.ErrStart {
		// TODO: update span status
		h.tracer.Finish(span)
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "%s", err.Error()))
		return
	} else if err != nil && err == sidecar.ErrComponentNotFound {
		// TODO: update span status
		h.tracer.Finish(span)
		httputils.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), secretId))
		return
	} else if err != nil {
		// TODO: update span status
		h.tracer.Finish(span)
		httputils.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve secret from store %s and key %s: %v", secretId, key, err))
		return
	}

	// TODO: update span status

	h.tracer.Finish(span)

	httputils.OkResponse(w, secret)
}

func NewSecretHandler(s sidecar.Sidecar, t tracev2.Trace) SecretHandler {
	return &secretHandler{s, t}
}
