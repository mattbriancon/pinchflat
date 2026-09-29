package web

import (
	"net/http"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// SearchControllerShow: show(conn, params)
func (s *Server) SearchControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		searchTerm = ""
	}

	searchResults, err := s.App.Search(ctx, searchTerm, 0)
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	if searchResults == nil {
		searchResults = []*store.MediaItem{}
	}

	s.Render(w, r, http.StatusOK, LayoutApp, SearchHTMLShow(searchTerm, searchResults))
}
