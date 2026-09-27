package web

// Port of lib/pinchflat_web/controllers/sources/source_live/source_enable_toggle.ex.
// The render/1 templ component lives in the companion .templ file (this
// LiveComponent has no separate .heex source; its markup is inline in the
// .ex file via ~H).

import (
	"net/http"
	"strconv"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// SourceEnableToggleUpdate is POST /sources/{id}/enabled: it persists the
// toggle and redirects back to the page it came from (STRATEGY.md decision
// 4: a plain form POST, not an htmx fragment update). Like the Elixir
// version, an update failure (e.g. a bad changeset) is silently ignored --
// Sources.update_source's result is never pattern-matched there either.
func (s *Server) SourceEnableToggleUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(URLParam(r, "id"), 10, 64)
	sourceParams := ParseForm(r, "source")

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	_, _ = s.App.SourcesUpdateSource(ctx, source, sourceParams, core.KW{})

	s.redirectBack(w, r, P(ctx, "/sources"))
}
