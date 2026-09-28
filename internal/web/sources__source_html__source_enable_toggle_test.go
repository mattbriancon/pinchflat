package web_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestSourceEnableToggle_InitialRendering(t *testing.T) {
	t.Run("renders a toggle in the on position if the source is enabled", func(t *testing.T) {
		html, err := templ.ToGoHTML(context.Background(), web.SourceEnableToggleRender(1, true))
		if err != nil {
			t.Fatalf("render failed: %v", err)
		}

		// This is checking the Alpine attrs which is a good-enough proxy for the toggle position
		if !strings.Contains(string(html), "{ enabled: true }") {
			t.Errorf("expected '{ enabled: true }' in response, got: %s", html)
		}
	})

	t.Run("renders a toggle in the off position if the source is disabled", func(t *testing.T) {
		html, err := templ.ToGoHTML(context.Background(), web.SourceEnableToggleRender(1, false))
		if err != nil {
			t.Fatalf("render failed: %v", err)
		}

		if !strings.Contains(string(html), "{ enabled: false }") {
			t.Errorf("expected '{ enabled: false }' in response, got: %s", html)
		}
	})
}

// Port of the LiveView test/pinchflat_web/controllers/sources/source_live/index_table_live_test.exs
// enable-toggle case: it's now a plain POST /sources/{id}/enabled that
// redirects back (STRATEGY.md decision 4: no htmx).
func TestSourceEnableToggle_Update(t *testing.T) {
	t.Run("updates the source's enabled status and redirects to /sources by default", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{"enabled": true})

		res := c.Post(fmt.Sprintf("/sources/%d/enabled", source.ID), "source", store.Attrs{"enabled": false})

		if res.Status != 303 {
			t.Fatalf("expected a 303, got %d", res.Status)
		}
		if got := res.Header.Get("Location"); got != "/sources" {
			t.Errorf("expected redirect to /sources, got %q", got)
		}

		reloaded, err := c.App.GetSource(c.Ctx, source.ID)
		if err != nil {
			t.Fatalf("reload failed: %v", err)
		}
		if reloaded.Enabled {
			t.Errorf("expected source to be disabled")
		}
	})

	t.Run("redirects back to a same-origin Referer under BASE_ROUTE_PATH", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{"enabled": true})

		req := httptest.NewRequest("POST", fmt.Sprintf("/sources/%d/enabled", source.ID), strings.NewReader("source[enabled]=false"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Referer", "http://example.com/sources/"+fmt.Sprint(source.ID))
		res := c.Do(req)

		if res.Status != 303 {
			t.Fatalf("expected a 303, got %d", res.Status)
		}
		if got := res.Header.Get("Location"); got != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect back to the referer, got %q", got)
		}
	})

	t.Run("ignores a cross-origin Referer", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{"enabled": true})

		req := httptest.NewRequest("POST", fmt.Sprintf("/sources/%d/enabled", source.ID), strings.NewReader("source[enabled]=false"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Referer", "http://evil.example/sources")
		res := c.Do(req)

		if res.Status != 303 {
			t.Fatalf("expected a 303, got %d", res.Status)
		}
		if got := res.Header.Get("Location"); got != "/sources" {
			t.Errorf("expected the fallback /sources, got %q", got)
		}
	})
}
