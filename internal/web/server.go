// Package web is the Go port of lib/pinchflat_web: router, plugs,
// controllers (handlers), templates (templ) and components.
package web

import (
	"context"
	"net/http"

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
	// Layout components (LayoutsRoot/LayoutsApp/LayoutsOnboarding, called by
	// render.go with a single templ.Component argument) and a few
	// CustomComponents (datetime_in_zone's timezone) read Settings/Config
	// the way the equivalent Elixir template did, but templ gives them no
	// channel for that beyond ctx. See currentApp/currentOpts in layouts.go.
	globalApp, globalOpts = app, opts
	return &Server{App: app, Opts: opts}
}

// ctx returns the request context (shorthand used by handlers).
func ctxOf(r *http.Request) context.Context { return r.Context() }
