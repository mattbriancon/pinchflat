package web_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

// Port of test/pinchflat_web/controllers/pages/history_table_live_test.exs.
// The history tables are now rendered inline on the home page ("/") instead
// of the old /_live/history fragment (STRATEGY.md decision 4: no htmx).

func downloadedMediaItem(t testing.TB, c *webtest.Client, customName string) *store.MediaItem {
	t.Helper()
	source := coretest.SourceFixture(t, c.TestApp, store.Attrs{"custom_name": customName})
	return coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID, "title": customName})
}

func pendingMediaItem(t testing.TB, c *webtest.Client, customName string) *store.MediaItem {
	t.Helper()
	source := coretest.SourceFixture(t, c.TestApp, store.Attrs{"custom_name": customName})
	return coretest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID, "title": customName, "media_filepath": nil})
}

func TestHistoryTableLive_InitialRendering(t *testing.T) {
	t.Run("shows message when no downloaded records", func(t *testing.T) {
		c := homeClient(t)
		html := c.Get("/").HTML(t, 200)

		if !strings.Contains(html, "Nothing Here!") {
			t.Errorf("expected 'Nothing Here!' in response")
		}
	})

	t.Run("shows downloaded records", func(t *testing.T) {
		c := homeClient(t)
		item := downloadedMediaItem(t, c, "Downloaded Item")

		html := c.Get("/").HTML(t, 200)
		if !strings.Contains(html, store.Deref(item.Title)) {
			t.Errorf("expected %q in response", store.Deref(item.Title))
		}
	})

	t.Run("shows pending records", func(t *testing.T) {
		c := homeClient(t)
		item := pendingMediaItem(t, c, "Pending Item")

		html := c.Get("/").HTML(t, 200)
		if !strings.Contains(html, store.Deref(item.Title)) {
			t.Errorf("expected %q in response", store.Deref(item.Title))
		}
	})
}

func TestHistoryTableLive_WhenTestingPagination(t *testing.T) {
	t.Run("paging the downloaded table doesn't reset the pending table's page", func(t *testing.T) {
		c := homeClient(t)
		var downloaded []*store.MediaItem
		for i := 0; i < 6; i++ {
			downloaded = append(downloaded, downloadedMediaItem(t, c, fmt.Sprintf("Downloaded_%02d", i)))
		}
		var pending []*store.MediaItem
		for i := 0; i < 6; i++ {
			pending = append(pending, pendingMediaItem(t, c, fmt.Sprintf("Pending_%02d", i)))
		}

		// Page 2 of "downloaded" and page 2 of "pending" show different,
		// independent items -- requesting one doesn't disturb the other's
		// first-page default.
		htmlDownloadedPage2 := c.Get("/", map[string]string{"downloaded_page": "2"}).HTML(t, 200)
		if !strings.Contains(htmlDownloadedPage2, store.Deref(pending[len(pending)-1].Title)) {
			t.Errorf("expected pending table to still show its first page")
		}

		htmlBothPage2 := c.Get("/", map[string]string{"downloaded_page": "2", "pending_page": "2"}).HTML(t, 200)
		if strings.Contains(htmlBothPage2, store.Deref(pending[len(pending)-1].Title)) {
			t.Errorf("expected pending table's page 2 to no longer show its most recent item")
		}
	})
}
