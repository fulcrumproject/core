// JWT discovery exposes only the server's public verification keys.
// This handler is mounted outside API authentication for offline verification.
package api

import (
	"net/http"

	"github.com/fulcrumproject/core/pkg/jwtissuer"
	"github.com/go-chi/render"
)

type JwksHandler struct {
	signer *jwtissuer.Signer
}

func NewJwksHandler(signer *jwtissuer.Signer) *JwksHandler {
	return &JwksHandler{signer: signer}
}

// JWKS returns an empty set when signing is completely unconfigured.
func (h *JwksHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, h.signer.JWKS())
}
