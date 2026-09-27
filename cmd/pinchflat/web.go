package main

import (
	"net/http"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/web"
)

func newHandler(app *core.App, s Settings) http.Handler {
	return web.New(app, web.Options{
		SecretKeyBase:    s.SecretKeyBase,
		EnablePrometheus: s.EnablePrometheus,
		Version:          version,
	}).Handler()
}
