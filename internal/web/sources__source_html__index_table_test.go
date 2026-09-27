package web_test

import (
	"fmt"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestSourceIndexTable_InitialRendering(t *testing.T) {
	t.Run("lists all sources", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Get("/sources")

		if !strings.Contains(res.HTML(t, 200), source.CustomName) {
			t.Errorf("expected %q in response", source.CustomName)
		}
	})

	t.Run("omits sources that have marked_for_deletion_at set", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{"marked_for_deletion_at": coretest.Now()})

		res := c.Get("/sources")

		if strings.Contains(res.HTML(t, 200), source.CustomName) {
			t.Errorf("did not expect %q in response", source.CustomName)
		}
	})

	t.Run("omits sources who's media profile has marked_for_deletion_at set", func(t *testing.T) {
		c := webtest.New(t)
		mediaProfile := coretest.MediaProfileFixture(t, c.TestApp, core.Attrs{"marked_for_deletion_at": coretest.Now()})
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{"media_profile_id": mediaProfile.ID})

		res := c.Get("/sources")

		if strings.Contains(res.HTML(t, 200), source.CustomName) {
			t.Errorf("did not expect %q in response", source.CustomName)
		}
	})

	t.Run("links each source's RSS feed", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		req := httptest.NewRequest("GET", "/sources", nil)
		req.Host = "pinchflat.example.com"
		req.Header.Set("X-Forwarded-Proto", "https")
		html := c.Do(req).HTML(t, 200)

		want := fmt.Sprintf(`href="https://pinchflat.example.com/sources/%s/feed.xml"`, core.Deref(source.UUID))
		if !strings.Contains(html, want) {
			t.Errorf("expected %s in response, got: %s", want, html)
		}
		if strings.Contains(html, "podcast://") {
			t.Errorf("did not expect a podcast:// link")
		}
	})

	t.Run("falls back to the endpoint URL for RSS feeds", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		html := c.Get("/sources").HTML(t, 200)

		if !strings.Contains(html, fmt.Sprintf("/sources/%s/feed.xml", core.Deref(source.UUID))) {
			t.Errorf("expected the feed path in response")
		}
	})
}

func TestSourceIndexTable_WhenTestingSorting(t *testing.T) {
	t.Run("sorts by the custom_name by default", func(t *testing.T) {
		c := webtest.New(t)
		source1 := coretest.SourceFixture(t, c.TestApp, core.Attrs{"custom_name": "Source_B"})
		source2 := coretest.SourceFixture(t, c.TestApp, core.Attrs{"custom_name": "Source_A"})

		html := c.Get("/sources").HTML(t, 200)

		AssertBefore(t, html, source2.CustomName, source1.CustomName)
	})

	t.Run("clicking the row will change the sort direction", func(t *testing.T) {
		c := webtest.New(t)
		source1 := coretest.SourceFixture(t, c.TestApp, core.Attrs{"custom_name": "Source_B"})
		source2 := coretest.SourceFixture(t, c.TestApp, core.Attrs{"custom_name": "Source_A"})

		// Clicking the "Name" header toggles custom_name asc -> desc.
		html := c.Get("/sources", map[string]string{"sort_key": "custom_name", "sort_direction": "desc"}).HTML(t, 200)

		AssertBefore(t, html, source1.CustomName, source2.CustomName)
	})

	t.Run("clicking a different row will sort by that attribute", func(t *testing.T) {
		c := webtest.New(t)
		source1 := coretest.SourceFixture(t, c.TestApp, core.Attrs{"custom_name": "Source_A", "enabled": true})
		source2 := coretest.SourceFixture(t, c.TestApp, core.Attrs{"custom_name": "Source_A", "enabled": false})

		// Clicking the "Enabled?" header: false sorts before true ascending.
		html := c.Get("/sources", map[string]string{"sort_key": "enabled", "sort_direction": "asc"}).HTML(t, 200)
		AssertBefore(t, html, sourceRowMarker(source2.ID), sourceRowMarker(source1.ID))

		// Clicking it again toggles the direction.
		html = c.Get("/sources", map[string]string{"sort_key": "enabled", "sort_direction": "desc"}).HTML(t, 200)
		AssertBefore(t, html, sourceRowMarker(source1.ID), sourceRowMarker(source2.ID))
	})

	t.Run("name is sorted without case sensitivity", func(t *testing.T) {
		c := webtest.New(t)
		source1 := coretest.SourceFixture(t, c.TestApp, core.Attrs{"custom_name": "Source_B"})
		source2 := coretest.SourceFixture(t, c.TestApp, core.Attrs{"custom_name": "source_a"})

		html := c.Get("/sources").HTML(t, 200)

		AssertBefore(t, html, source2.CustomName, source1.CustomName)
	})
}

func TestSourceIndexTable_WhenTestingPagination(t *testing.T) {
	t.Run("moving to the next page loads new records", func(t *testing.T) {
		c := webtest.New(t)
		// The Go port hardcodes the LiveView's `results_per_page: 10` (there's
		// no LiveView session to override it per-test), so eleven sources are
		// used here instead of the Elixir test's results_per_page: 1 override.
		var sources []*core.Source
		for i := 0; i < 11; i++ {
			sources = append(sources, coretest.SourceFixture(t, c.TestApp, core.Attrs{
				"custom_name": fmt.Sprintf("Source_%02d", i),
			}))
		}
		first, last := sources[0], sources[10]

		html := c.Get("/sources").HTML(t, 200)
		if !strings.Contains(html, first.CustomName) {
			t.Errorf("expected first page to contain %q", first.CustomName)
		}
		if strings.Contains(html, last.CustomName) {
			t.Errorf("did not expect first page to contain %q", last.CustomName)
		}

		html = c.Get("/sources", map[string]string{"page": "2"}).HTML(t, 200)
		if strings.Contains(html, first.CustomName) {
			t.Errorf("did not expect second page to contain %q", first.CustomName)
		}
		if !strings.Contains(html, last.CustomName) {
			t.Errorf("expected second page to contain %q", last.CustomName)
		}
	})
}

// TestSourceIndexTable_WhenTestingTheEnableToggle moved to
// sources__source_html__source_enable_toggle_test.go: enabling/disabling a
// source is now its own POST /sources/{id}/enabled route (see router.go).

func AssertBefore(t *testing.T, html, first, second string) {
	t.Helper()
	i, j := strings.Index(html, first), strings.Index(html, second)
	if i < 0 {
		t.Fatalf("expected %q in response", first)
	}
	if j < 0 {
		t.Fatalf("expected %q in response", second)
	}
	if i >= j {
		t.Errorf("expected %q to appear before %q", first, second)
	}
}

func sourceRowMarker(id int64) string {
	return "/sources/" + strconv.FormatInt(id, 10) + "\""
}
