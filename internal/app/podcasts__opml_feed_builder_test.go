package app_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestOpmlFeedBuilder_Build(t *testing.T) {
	ta := apptest.NewApp(t)

	source := apptest.SourceFixture(t, ta, store.SourceParams{})

	t.Run("returns an XML document", func(t *testing.T) {
		res := app.OpmlFeedBuilderBuild("http://example.com", []*store.Source{source})

		if !strings.Contains(res, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Errorf("expected XML declaration, got: %v", res)
		}
	})

	t.Run("escapes illegal characters", func(t *testing.T) {
		source := apptest.SourceFixture(t, ta, store.SourceParams{CustomName: store.Ptr("A & B")})
		res := app.OpmlFeedBuilderBuild("http://example.com", []*store.Source{source})

		if !strings.Contains(res, `A &amp; B`) {
			t.Errorf("expected escaped ampersand, got: %v", res)
		}
	})

	t.Run("build podcast link with URL base", func(t *testing.T) {
		res := app.OpmlFeedBuilderBuild("http://example.com", []*store.Source{source})

		expectedURL := "http://example.com/sources/" + *source.UUID + "/feed.xml"
		if !strings.Contains(res, expectedURL) {
			t.Errorf("expected %q in output, got: %v", expectedURL, res)
		}
	})
}
