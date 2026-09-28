package web

// Small helpers factoring out patterns repeated across the resource
// controllers (sources, media items, media profiles): parsing a route
// param as an id, and loading a record or failing the request.

import (
	"context"
	"net/http"
	"strconv"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// parseIDParam parses the named route param as an int64, or returns
// (0, false) when it's missing or not a valid id.
func parseIDParam(r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(URLParam(r, param), 10, 64)
	return id, err == nil
}

// loadOrFail parses the named route param as an id and fetches the record
// with get, writing the appropriate 404/500 response via s.Fail on any
// failure. The caller should return immediately when ok is false.
func loadOrFail[T any](s *Server, w http.ResponseWriter, r *http.Request, param string, get func(context.Context, int64) (T, error)) (v T, ok bool) {
	id, valid := parseIDParam(r, param)
	if !valid {
		s.Fail(w, r, store.ErrNotFound)
		return v, false
	}
	v, err := get(r.Context(), id)
	if err != nil {
		s.Fail(w, r, err)
		return v, false
	}
	return v, true
}
