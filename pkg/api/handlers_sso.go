// SSO discovery exposes only the server's public verification keys.
// This handler is mounted outside API authentication for offline verification.
package api

import (
	"net/http"

	"github.com/fulcrumproject/core/pkg/sso"
	"github.com/go-chi/render"
)

type SsoHandler struct {
	signer *sso.Signer
}

func NewSsoHandler(signer *sso.Signer) *SsoHandler {
	return &SsoHandler{signer: signer}
}

// JWKS returns an empty set when signing is completely unconfigured.
func (h *SsoHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, h.signer.JWKS())
}
