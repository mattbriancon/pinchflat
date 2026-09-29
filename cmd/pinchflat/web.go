package main

import (
	"net/http"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/web"
)

func newHandler(app *app.App, s Settings) http.Handler {
	return web.New(app, web.Options{
		SecretKeyBase:    s.SecretKeyBase,
		EnablePrometheus: s.EnablePrometheus,
		Version:          version,
	}).Handler()
}
