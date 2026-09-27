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

	t.Run("returns XML", func(t *testing.T) {

		res := core.OpmlFeedBuilderBuild("http://example.com", []*core.Source{source})

		if !strings.Contains(res, `<?xml version="1.0" encoding="UTF-8"?>`) {
			t.Errorf("missing XML declaration")
		}
	})

	t.Run("escapes ampersands", func(t *testing.T) {

		src := coretest.SourceFixture(t, ta, core.Attrs{"custom_name": "A & B"})
		res := core.OpmlFeedBuilderBuild("http://example.com", []*core.Source{src})

		if !strings.Contains(res, `A &amp; B`) {
			t.Errorf("missing escaped ampersand")
		}
	})

	t.Run("builds source link", func(t *testing.T) {

		res := core.OpmlFeedBuilderBuild("http://example.com", []*core.Source{source})

		expectedURL := "http://example.com/sources/" + *source.UUID + "/feed.xml"
		if !strings.Contains(res, expectedURL) {
			t.Errorf("missing source URL")
		}
	})
}
