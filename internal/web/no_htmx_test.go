package web_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

// htmxAttr matches a real htmx attribute (hx-get="...", hx-post="...", ...),
// not merely the substring "hx-" (which also occurs inside the unrelated,
// harmless Phoenix leftover attribute "phx-track-static").
var htmxAttr = regexp.MustCompile(`\bhx-[a-z-]+=`)

// TestNoHTMX scans several representative pages to make sure htmx is truly
// gone (STRATEGY.md decision 4): every page renders everything up front, so
// there's nothing left for htmx to fetch.
func TestNoHTMX(t *testing.T) {
	c := webtest.New(t)
	if _, err := c.App.SettingsSet(c.Ctx, core.KW{core.Opt("onboarding", false)}); err != nil {
		t.Fatalf("SettingsSet: %v", err)
	}
	source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
	coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID})
	coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID, "media_filepath": nil})

	pages := map[string]string{
		"home":          "/",
		"sources index": "/sources",
		"source show":   "/sources/" + strconv.FormatInt(source.ID, 10),
		"settings show": "/settings",
	}

	for name, path := range pages {
		t.Run(name, func(t *testing.T) {
			html := c.Get(path).HTML(t, 200)
			if loc := htmxAttr.FindString(html); loc != "" {
				t.Errorf("expected no htmx attributes on %s, found %q", path, loc)
			}
		})
	}
}
