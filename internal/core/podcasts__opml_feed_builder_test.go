package core_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestOpmlFeedBuilder_Build(t *testing.T) {
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	source := coretest.SourceFixture(t, ta, core.Attrs{})

	t.Run("returns an XML document", func(t *testing.T) {
		res := core.OpmlFeedBuilderBuild("http://example.com", []*core.Source{source})

		if !strings.Contains(res, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Errorf("expected XML declaration, got: %v", res)
		}
	})

	t.Run("escapes illegal characters", func(t *testing.T) {
		source := coretest.SourceFixture(t, ta, core.Attrs{"custom_name": "A & B"})
		res := core.OpmlFeedBuilderBuild("http://example.com", []*core.Source{source})

		if !strings.Contains(res, `A &amp; B`) {
			t.Errorf("expected escaped ampersand, got: %v", res)
		}
	})

	t.Run("build podcast link with URL base", func(t *testing.T) {
		res := core.OpmlFeedBuilderBuild("http://example.com", []*core.Source{source})

		expectedURL := "http://example.com/sources/" + *source.UUID + "/feed.xml"
		if !strings.Contains(res, expectedURL) {
			t.Errorf("expected %q in output, got: %v", expectedURL, res)
		}
	})
}
