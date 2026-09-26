package web_test

// Port of test/pinchflat_web/controllers/error_html_test.exs.
//
// error_json.ex was dropped from the manifest (no Go target), and the Go
// web layer has no JSON error renderer (Fail/NotFound in render.go always
// render the HTML error pages, never JSON): there is no equivalent surface
// to port test/pinchflat_web/controllers/error_json_test.exs against, so
// its two tests are skipped as DROPPED below instead of silently dropped.

import (
	"context"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/web"
)

func TestErrorHTML(t *testing.T) {
	t.Run("renders 404.html", func(t *testing.T) {
		var b strings.Builder
		if err := web.ErrorHTML404().Render(context.Background(), &b); err != nil {
			t.Fatalf("render failed: %v", err)
		}
		if !strings.Contains(b.String(), "404 (not found)") {
			t.Errorf("expected the body to contain %q, got %q", "404 (not found)", b.String())
		}
	})

	t.Run("renders 500.html", func(t *testing.T) {
		var b strings.Builder
		if err := web.ErrorHTML500().Render(context.Background(), &b); err != nil {
			t.Fatalf("render failed: %v", err)
		}
		if !strings.Contains(b.String(), "Internal Server Error") {
			t.Errorf("expected the body to contain %q, got %q", "Internal Server Error", b.String())
		}
	})
}

// Port of test/pinchflat_web/controllers/error_json_test.exs. error_json.ex
// was dropped from the manifest (the Go web layer never renders a JSON
// error body; render.go's Fail/NotFound always render the HTML error
// pages), so there is nothing to assert against.
func TestErrorJSON(t *testing.T) {
	t.Run("renders 404", func(t *testing.T) {
		t.Skip("DROPPED: error_json.ex has no Go target (manifest drops it); the Go web layer never renders a JSON error body")
	})

	t.Run("renders 500", func(t *testing.T) {
		t.Skip("DROPPED: error_json.ex has no Go target (manifest drops it); the Go web layer never renders a JSON error body")
	})
}
