package web_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

// htmxAttr matches an htmx or LiveView attribute (hx-get=, phx-click=, ...).
var htmxAttr = regexp.MustCompile(`\bp?hx-[a-z-]+=`)

// literalComponent matches a templ component call that was rendered as text
// (text and @Component(...) on the same templ line).
var literalComponent = regexp.MustCompile(`@[A-Z][A-Za-z]+\(`)

// TestNoHTMX scans several representative pages to make sure htmx is truly
// gone (STRATEGY.md decision 4): every page renders everything up front, so
// there's nothing left for htmx to fetch.
func TestNoHTMX(t *testing.T) {
	c := webtest.New(t)
	if _, err := c.App.SetSetting(c.Ctx, store.KW{store.Opt("onboarding", false)}); err != nil {
		t.Fatalf("SettingsSet: %v", err)
	}
	source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})
	apptest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID})
	apptest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID, "media_filepath": nil})

	pages := map[string]string{
		"home":          "/",
		"sources index": "/sources",
		"source show":   "/sources/" + strconv.FormatInt(source.ID, 10),
		"settings show": "/settings",
		"media profile": "/media_profiles",
		"app info":      "/app_info",
	}

	for name, path := range pages {
		t.Run(name, func(t *testing.T) {
			html := c.Get(path).HTML(t, 200)
			if loc := htmxAttr.FindString(html); loc != "" {
				t.Errorf("expected no htmx attributes on %s, found %q", path, loc)
			}
			if loc := literalComponent.FindString(html); loc != "" {
				t.Errorf("a templ component call rendered as text on %s: %q", path, loc)
			}
		})
	}
}
