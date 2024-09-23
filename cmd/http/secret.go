package http

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/w-h-a/pkg/sidecar"
	"github.com/w-h-a/pkg/utils/errorutils"
)

type SecretHandler interface {
	HandleGet(w http.ResponseWriter, r *http.Request)
}

type secretHandler struct {
	service sidecar.Sidecar
}

func (h *secretHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	secretId := params["secretId"]

	key := params["key"]

	secret, err := h.service.ReadFromSecretStore(secretId, key)
	if err != nil && err == sidecar.ErrComponentNotFound {
		ErrResponse(w, errorutils.NotFound("sidecar", "%s: %s", err.Error(), secretId))
		return
	} else if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to retrieve secret from store %s and key %s: %v", secretId, key, err))
		return
	}

	bs, err := json.Marshal(secret)
	if err != nil {
		ErrResponse(w, errorutils.InternalServerError("sidecar", "failed to marshal secret: %v", err))
		return
	}

	w.Header().Set("content-type", "application/json")
	w.WriteHeader(200)
	w.Write(bs)
}

func NewSecretHandler(s sidecar.Sidecar) SecretHandler {
	return &secretHandler{s}
}
