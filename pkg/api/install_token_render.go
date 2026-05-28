package api

import (
	"encoding/json"
	"net/http"
)

// writeInstallTokenJSON writes the install-token response without HTML
// escaping so installCommand stays copy-pasteable: Go's default json.Marshal
// would write `&` (and other HTML-significant chars) as `&`. Decoders
// parse the escape correctly, but the installCommand field is meant to be
// eyeballed and pasted into a shell straight from the response, where the
// literal `&` survives and breaks curl. Shared by the per-entity install
// handlers so neither hand-rolls the encoder.
func writeInstallTokenJSON(w http.ResponseWriter, status int, body InstallTokenRes) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(body)
}
