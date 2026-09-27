// Package web is the HTTP layer: router, plugs, controllers (handlers),
// templates (templ) and components.
package web

import (
	"github.com/mattbriancon/pinchflat/internal/core"
)

// Options are the web-only settings (PinchflatWeb.Endpoint config).
type Options struct {
	// SecretKeyBase signs the session/flash cookie.
	SecretKeyBase string
	// EnablePrometheus serves /metrics.
	EnablePrometheus bool
	// Version is shown on the app info page.
	Version string
}

// Server holds what handlers need. Handlers are methods on *Server named
// after the Elixir controller and action, e.g. SourceControllerIndex.
type Server struct {
	App  *core.App
	Opts Options
}

// New builds a Server.
func New(app *core.App, opts Options) *Server {
	if opts.SecretKeyBase == "" {
		opts.SecretKeyBase = "development-only-secret-key-base"
	}
	return &Server{App: app, Opts: opts}
}
