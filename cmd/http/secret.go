package http

import (
	gohttp "net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/serverv2/http"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/utils/errorutils"
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

	secret, err := h.service.ReadFromSecretStore(secretId, key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		http.ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), secretId))
		return
	} else if err != nil {
		http.ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve secret from store %s and key %s: %v", secretId, key, err))
		return
	}

	http.OkResponse(w, secret)
}

func NewSecretHandler(s sidecar.Sidecar) SecretHandler {
	return &secretHandler{s}
}
