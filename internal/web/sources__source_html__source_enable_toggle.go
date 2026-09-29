package web

import (
	"net/http"
)

// SourceEnableToggleUpdate is POST /sources/{id}/enabled: it persists the
// toggle and redirects back to the page it came from (a plain form POST,
// not a fragment update). An update failure (e.g. a bad changeset) is
// silently ignored; the toggle always redirects back.
func (s *Server) SourceEnableToggleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sourceParams := ParseForm(r, "source")

	source, ok := loadOrFail(s, w, r, "id", s.App.GetSource)
	if !ok {
		return
	}

	_, _ = s.App.SourcesUpdateSource(ctx, source, sourceParams, true)

	s.redirectBack(w, r, P(ctx, "/sources"))
}
