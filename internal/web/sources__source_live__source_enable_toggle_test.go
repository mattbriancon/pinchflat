package web_test

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/web"
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
