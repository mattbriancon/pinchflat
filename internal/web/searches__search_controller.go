package web

import (
	"net/http"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// Port of lib/pinchflat_web/controllers/searches/search_controller.ex.

// SearchControllerShow: show(conn, params)
func (s *Server) SearchControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		searchTerm = ""
	}

	searchResults, err := s.App.MediaSearch(ctx, searchTerm, core.KW{})
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	if searchResults == nil {
		searchResults = []*core.MediaItem{}
	}

	s.Render(w, r, http.StatusOK, LayoutApp, SearchHTMLShow(searchTerm, searchResults))
}
