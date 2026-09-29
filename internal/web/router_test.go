package web_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestRouter(t *testing.T) {
	c := webtest.New(t)
	for _, tc := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/", 200},
		{"GET", "/healthcheck", 200},
		{"GET", "/media_profiles/new", 200},
		{"GET", "/nope", 404},
		{"GET", "/sources/new/extra", 404},
		{"PATCH", "/media_profiles/new", 404}, // wrong method is a 404, not a 405
		{"POST", "/healthcheck", 404},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(""))
		if res := c.Do(req); res.Status != tc.status {
			t.Errorf("%s %s: expected %d, got %d", tc.method, tc.path, tc.status, res.Status)
		}
	}
}
