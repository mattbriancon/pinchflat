package web

// Per-request page state that Phoenix kept in conn assigns: flash, current
// path, query params, base path. Templates read it from ctx with PageOf.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// Page is available to every template via PageOf(ctx).
type Page struct {
	Request *http.Request
	// BasePath is BASE_ROUTE_PATH without a trailing slash ("" for "/").
	BasePath string
	// BaseURL is scheme://host[:port] as seen by the client (honours
	// x-forwarded-proto), used for absolute URLs such as podcast feeds.
	BaseURL string
	// Flash messages for this render (kind -> message).
	Flash map[string]string
	// Onboarding mirrors Settings.onboarding for layout choices.
	Onboarding bool
	// App and Version are the serving Server's, for layouts and components
	// that read Settings/Config (templ only hands them ctx).
	App     *core.App
	Version string
}

type pageKey struct{}

// PageOf returns the Page for this request (never nil).
func PageOf(ctx context.Context) *Page {
	if p, ok := ctx.Value(pageKey{}).(*Page); ok {
		return p
	}
	return &Page{Flash: map[string]string{}}
}

func withPage(r *http.Request, p *Page) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), pageKey{}, p))
}

// P builds an app path like Phoenix's ~p sigil: it prefixes BASE_ROUTE_PATH.
// Arguments are formatted with %v and path-escaped:
//
//	web.P(ctx, "/sources/%v/media/%v", source.ID, item.ID)
func P(ctx context.Context, format string, args ...any) string {
	escaped := make([]any, len(args))
	for i, a := range args {
		escaped[i] = url.PathEscape(fmt.Sprint(a))
	}
	p := fmt.Sprintf(format, escaped...)
	return PageOf(ctx).BasePath + p
}

// URL is the absolute form of P (Phoenix's url(~p"...")).
func URL(ctx context.Context, format string, args ...any) string {
	return PageOf(ctx).BaseURL + P(ctx, format, args...)
}

// CurrentPath is Phoenix.Controller.current_path(conn, %{}): the request
// path without query string, including the base path.
func CurrentPath(ctx context.Context) string {
	if r := PageOf(ctx).Request; r != nil {
		return PageOf(ctx).BasePath + r.URL.Path
	}
	return ""
}

// Param returns a query parameter (@conn.params["q"]).
func Param(ctx context.Context, name string) string {
	if r := PageOf(ctx).Request; r != nil {
		return r.URL.Query().Get(name)
	}
	return ""
}

func trimBase(p string) string {
	p = strings.TrimSuffix(p, "/")
	if p == "" || p == "/" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}
