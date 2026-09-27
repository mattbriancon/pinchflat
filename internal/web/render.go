package web

// Rendering helpers: the Go side of Phoenix's render/3, redirect/2, json/2
// and put_status. Hand-written W4 infrastructure.

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/core"
)

// Layout selects the inner layout (the root layout always wraps it).
type Layout int

const (
	LayoutApp        Layout = iota // layouts/app.html.heex (default)
	LayoutOnboarding               // layouts/onboarding.html.heex
	LayoutNone                     // fragments (htmx) and error pages
)

// OnboardingLayout is get_onboarding_layout/0: the onboarding layout while
// Settings.onboarding is true, else the app layout.
func OnboardingLayout(ctx context.Context) Layout {
	if PageOf(ctx).Onboarding {
		return LayoutOnboarding
	}
	return LayoutApp
}

// Render writes a full page: root layout > layout > content.
func (s *Server) Render(w http.ResponseWriter, r *http.Request, status int, layout Layout, content templ.Component) {
	page := content
	switch layout {
	case LayoutApp:
		page = LayoutsApp(content)
	case LayoutOnboarding:
		page = LayoutsOnboarding(content)
	}
	if layout != LayoutNone {
		page = LayoutsRoot(page)
	}
	s.write(w, r, status, page)
}

func (s *Server) write(w http.ResponseWriter, r *http.Request, status int, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := c.Render(r.Context(), w); err != nil {
		slog.Error("render failed", "path", r.URL.Path, "err", err)
	}
}

// Redirect is redirect(conn, to: path) (302). path must already include the
// base path (use P).
func (s *Server) Redirect(w http.ResponseWriter, r *http.Request, path string) {
	http.Redirect(w, r, path, http.StatusFound)
}

// JSON is json(conn, data).
func (s *Server) JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// NotFound renders the 404 page (ErrorHTML 404).
func (s *Server) NotFound(w http.ResponseWriter, r *http.Request) {
	s.Render(w, r, http.StatusNotFound, LayoutNone, ErrorHTML404())
}

// Fail maps an error to a response the way Phoenix does: a missing record
// (Ecto.NoResultsError / core.ErrNotFound) is a 404, anything else a 500.
func (s *Server) Fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, core.ErrNotFound) {
		s.NotFound(w, r)
		return
	}
	slog.Error("request failed", "path", r.URL.Path, "err", err)
	s.Render(w, r, http.StatusInternalServerError, LayoutNone, ErrorHTML500())
}
