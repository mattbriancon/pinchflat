package main

import (
	"net/http"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// newHandler builds the HTTP handler. Until the web layer (W4) is ported
// only the healthcheck is served.
func newHandler(app *core.App, s Settings) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthcheck", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(`{"status":"ok"}`))
	})
	return mux
}
