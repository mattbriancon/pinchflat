package web

// redirectBack is the "reload the page we came from" pattern the no-htmx
// mutations use (the source enable toggle): a 303 back to the
// Referer when it's same-origin and under BASE_ROUTE_PATH, else a sensible
// fallback (STRATEGY.md decision 4).

import (
	"net/http"
	"net/url"
	"strings"
)

// redirectBack sends a 303 to the Referer (path+query) when it's this app's
// own origin and its path is under BASE_ROUTE_PATH, else to fallback
// (already including the base path; use P).
func (s *Server) redirectBack(w http.ResponseWriter, r *http.Request, fallback string) {
	loc := fallback
	if ref := r.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil {
			page := PageOf(r.Context())
			if u.Scheme+"://"+u.Host == page.BaseURL && strings.HasPrefix(u.Path, page.BasePath) {
				loc = u.Path
				if u.RawQuery != "" {
					loc += "?" + u.RawQuery
				}
			}
		}
	}
	http.Redirect(w, r, loc, http.StatusSeeOther)
}
