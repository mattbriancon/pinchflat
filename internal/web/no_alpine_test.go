package web_test

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

// alpineMarkup matches Alpine.js directives and its script.
var alpineMarkup = regexp.MustCompile(`x-data|x-on:|x-bind:|x-show|x-model|x-text|x-init|x-cloak|@click|(?i:alpine)`)

// TestNoAlpine makes sure no page uses or loads Alpine.js (STRATEGY.md §11:
// pages use native HTML plus web/assets/js/app.js).
func TestNoAlpine(t *testing.T) {
	c := webtest.New(t)
	if _, err := c.App.SetSetting(c.Ctx, "onboarding", false); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})
	source := apptest.SourceFixture(t, c.TestApp, store.SourceParams{})
	item := apptest.MediaItemFixture(t, c.TestApp, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
	sid := strconv.FormatInt(source.ID, 10)

	pages := map[string]string{
		"home":               "/",
		"sources index":      "/sources",
		"source show":        "/sources/" + sid,
		"source new":         "/sources/new",
		"source edit":        "/sources/" + sid + "/edit",
		"media item show":    "/sources/" + sid + "/media/" + strconv.FormatInt(item.ID, 10),
		"media item edit":    "/sources/" + sid + "/media/" + strconv.FormatInt(item.ID, 10) + "/edit",
		"media profiles":     "/media_profiles",
		"media profile new":  "/media_profiles/new",
		"media profile edit": "/media_profiles/" + strconv.FormatInt(profile.ID, 10) + "/edit",
		"media profile show": "/media_profiles/" + strconv.FormatInt(profile.ID, 10),
		"settings":           "/settings",
		"app info":           "/app_info",
	}

	for name, path := range pages {
		t.Run(name, func(t *testing.T) {
			html := c.Get(path).HTML(t, 200)
			if loc := alpineMarkup.FindString(html); loc != "" {
				t.Errorf("expected no Alpine.js on %s, found %q", path, loc)
			}
		})
	}
}
