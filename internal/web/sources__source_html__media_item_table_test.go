package web_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

// Port of test/pinchflat_web/controllers/sources/source_html/media_item_table_live_test.exs.
// All three media_state tables (pending/downloaded/other) are now rendered
// inline on GET /sources/{id} (STRATEGY.md decision 4: no htmx, no
// media_state query param -- each tab renders its own state at once), so
// these tests fetch the source show page and scope assertions to the tab
// section they're about.

// mediaSectionHTML returns the slice of html for one media_state tab's
// wrapper (its content, up to the next media table's wrapper or the end of
// the response).
func mediaSectionHTML(t *testing.T, html string, sourceID int64, mediaState string) string {
	t.Helper()
	marker := fmt.Sprintf("id=\"media-table-%d-%s\"", sourceID, mediaState)
	start := strings.Index(html, marker)
	if start < 0 {
		t.Fatalf("expected %q in response, got: %s", marker, html)
	}
	rest := html[start+len(marker):]
	if idx := strings.Index(rest, "id=\"media-table-"); idx >= 0 {
		rest = rest[:idx]
	}
	return rest
}

func TestMediaItemTableLive_InitialRendering(t *testing.T) {
	t.Run("shows message when no records", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{})

		html := c.Get(fmt.Sprintf("/sources/%d", source.ID)).HTML(t, 200)
		pending := mediaSectionHTML(t, html, source.ID, "pending")

		if !strings.Contains(pending, "Nothing Here!") {
			t.Errorf("expected 'Nothing Here!' in response")
		}
		if strings.Contains(pending, "Showing") {
			t.Errorf("did not expect 'Showing' in response")
		}
	})

	t.Run("shows records when present", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{})
		mediaItem := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID, "media_filepath": nil})

		html := c.Get(fmt.Sprintf("/sources/%d", source.ID)).HTML(t, 200)
		pending := mediaSectionHTML(t, html, source.ID, "pending")

		if !strings.Contains(pending, "Showing") {
			t.Errorf("expected 'Showing' in response")
		}
		if !strings.Contains(pending, "Title") {
			t.Errorf("expected 'Title' in response")
		}
		if !strings.Contains(pending, store.Deref(mediaItem.Title)) {
			t.Errorf("expected media item title in response")
		}
	})
}

func TestMediaItemTableLive_MediaState(t *testing.T) {
	t.Run("shows pending media when pending", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{})
		downloadedMediaItem := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID})
		pendingMediaItem := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID, "media_filepath": nil})

		html := c.Get(fmt.Sprintf("/sources/%d", source.ID)).HTML(t, 200)
		pending := mediaSectionHTML(t, html, source.ID, "pending")

		if !strings.Contains(pending, store.Deref(pendingMediaItem.Title)) {
			t.Errorf("expected pending media item title in response")
		}
		if strings.Contains(pending, store.Deref(downloadedMediaItem.Title)) {
			t.Errorf("did not expect downloaded media item title in pending section")
		}
	})

	t.Run("shows downloaded media when downloaded", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{})
		downloadedMediaItem := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID})
		pendingMediaItem := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID, "media_filepath": nil})

		html := c.Get(fmt.Sprintf("/sources/%d", source.ID)).HTML(t, 200)
		downloaded := mediaSectionHTML(t, html, source.ID, "downloaded")

		if !strings.Contains(downloaded, store.Deref(downloadedMediaItem.Title)) {
			t.Errorf("expected downloaded media item title in response")
		}
		if strings.Contains(downloaded, store.Deref(pendingMediaItem.Title)) {
			t.Errorf("did not expect pending media item title in downloaded section")
		}
	})

	t.Run("shows records that aren't pending or downloaded when other", func(t *testing.T) {
		c := webtest.New(t)
		mediaProfile := coretest.MediaProfileFixture(t, c.TestApp, store.Attrs{"shorts_behaviour": "exclude"})
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{"media_profile_id": mediaProfile.ID})

		downloadedMediaItem := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID})
		pendingMediaItem := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID, "media_filepath": nil})
		otherMediaItem := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{
			"source_id":          source.ID,
			"media_filepath":     nil,
			"short_form_content": true,
		})

		html := c.Get(fmt.Sprintf("/sources/%d", source.ID)).HTML(t, 200)
		other := mediaSectionHTML(t, html, source.ID, "other")

		if !strings.Contains(other, store.Deref(otherMediaItem.Title)) {
			t.Errorf("expected other media item title in response")
		}
		if strings.Contains(other, store.Deref(downloadedMediaItem.Title)) {
			t.Errorf("did not expect downloaded media item title in other section")
		}
		if strings.Contains(other, store.Deref(pendingMediaItem.Title)) {
			t.Errorf("did not expect pending media item title in other section")
		}
	})

	t.Run("shows 'Manually Ignored' column when other", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{})
		coretest.MediaItemFixture(t, c.TestApp, store.Attrs{
			"source_id":        source.ID,
			"prevent_download": true,
			"media_filepath":   nil,
		})

		html := c.Get(fmt.Sprintf("/sources/%d", source.ID)).HTML(t, 200)
		other := mediaSectionHTML(t, html, source.ID, "other")

		if !strings.Contains(other, "Manually Ignored?") {
			t.Errorf("expected 'Manually Ignored?' in response")
		}
	})
}

func TestMediaItemTableLive_Search(t *testing.T) {
	t.Run("q filters records and stays on the matching tab", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{})
		match := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID, "media_filepath": nil, "title": "Match Me"})
		other := coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID, "media_filepath": nil, "title": "Something Else"})

		html := c.Get(fmt.Sprintf("/sources/%d", source.ID), map[string]string{"pending_q": "Match"}).HTML(t, 200)
		pending := mediaSectionHTML(t, html, source.ID, "pending")

		if !strings.Contains(pending, store.Deref(match.Title)) {
			t.Errorf("expected matching item in response")
		}
		if strings.Contains(pending, store.Deref(other.Title)) {
			t.Errorf("did not expect non-matching item in response")
		}
		if !strings.Contains(html, "value=\"Match\"") {
			t.Errorf("expected the search box to keep the typed value, got: %s", html)
		}
	})
}

func TestMediaItemTableLive_Pagination(t *testing.T) {
	t.Run("paging one tab doesn't reset another's page", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, store.Attrs{})
		var pendingItems []*store.MediaItem
		for i := 0; i < 11; i++ {
			pendingItems = append(pendingItems, coretest.MediaItemFixture(t, c.TestApp, store.Attrs{
				"source_id": source.ID, "media_filepath": nil, "title": fmt.Sprintf("Pending_%02d", i),
			}))
		}
		var downloadedItems []*store.MediaItem
		for i := 0; i < 11; i++ {
			downloadedItems = append(downloadedItems, coretest.MediaItemFixture(t, c.TestApp, store.Attrs{
				"source_id": source.ID, "title": fmt.Sprintf("Downloaded_%02d", i),
			}))
		}

		html := c.Get(fmt.Sprintf("/sources/%d", source.ID), map[string]string{"downloaded_page": "2"}).HTML(t, 200)
		pending := mediaSectionHTML(t, html, source.ID, "pending")

		// Records are ordered by uploaded_at DESC, which (fixtures created
		// within the same second) ties back to insertion order, so the
		// first-created item is on page 1.
		if !strings.Contains(pending, store.Deref(pendingItems[0].Title)) {
			t.Errorf("expected pending's first page to be unaffected by downloaded's page")
		}
		_ = downloadedItems
	})
}
