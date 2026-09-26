package web_test

// Port of test/pinchflat_web/controllers/media_item_controller_test.exs

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestMediaItemController_Show(t *testing.T) {
	t.Run("show media", func(t *testing.T) {
		t.Run("renders the page", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{})

			res := c.Get(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))
			html := res.HTML(t, 200)

			title := ""
			if mediaItem.Title != nil {
				title = *mediaItem.Title
			}
			if title != "" && !strings.Contains(html, title) {
				t.Errorf("expected to find media title in response")
			}
		})

		t.Run("renders the page when the media item has no description", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemWithAttachmentsFixture(t, c.TestApp, core.Attrs{"description": nil})

			res := c.Get(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))
			html := res.HTML(t, 200)

			title := ""
			if mediaItem.Title != nil {
				title = *mediaItem.Title
			}
			if title != "" && !strings.Contains(html, title) {
				t.Errorf("expected to find media title in response")
			}
		})
	})
}

func TestMediaItemController_Edit(t *testing.T) {
	t.Run("edit media", func(t *testing.T) {
		t.Run("renders form for editing chosen media_item", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{})

			res := c.Get(fmt.Sprintf("/sources/%d/media/%d/edit", mediaItem.SourceID, mediaItem.ID))
			html := res.HTML(t, 200)

			if !strings.Contains(html, "Editing") {
				t.Errorf("expected to find 'Editing' in response")
			}
		})
	})
}

func TestMediaItemController_Update(t *testing.T) {
	t.Run("update media", func(t *testing.T) {
		t.Run("redirects when data is valid", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{})

			updateAttrs := core.Attrs{"title": "New Title"}
			res := c.Patch(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID), "media_item", updateAttrs)

			expected := fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID)
			if redirected := res.RedirectedTo(t); redirected != expected {
				t.Errorf("expected redirect to %q, got %q", expected, redirected)
			}

			res = c.Get(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))
			html := res.HTML(t, 200)

			if !strings.Contains(html, "New Title") {
				t.Errorf("expected to find 'New Title' in response")
			}
		})

		t.Run("renders errors when data is invalid", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{})

			res := c.Patch(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID), "media_item", core.Attrs{"title": nil})
			html := res.HTML(t, 200)

			if !strings.Contains(html, "Editing") {
				t.Errorf("expected to find 'Editing' in response")
			}
		})
	})
}

func TestMediaItemController_Delete(t *testing.T) {
	t.Run("delete media", func(t *testing.T) {
		t.Run("the media item not is deleted", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemWithAttachmentsFixture(t, c.TestApp, core.Attrs{})

			c.Delete(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))

			reloaded, _ := c.App.MediaGetMediaItem(c.Ctx, mediaItem.ID)
			if reloaded == nil {
				t.Errorf("expected media item to still exist")
			}
		})

		t.Run("the files are deleted", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemWithAttachmentsFixture(t, c.TestApp, core.Attrs{})

			filepath := ""
			if mediaItem.MediaFilepath != nil {
				filepath = *mediaItem.MediaFilepath
			}

			c.Delete(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))

			if filepath != "" {
				if _, err := os.Stat(filepath); err == nil {
					t.Errorf("expected media file to be deleted")
				}
			}
		})

		t.Run("redirects to the source page", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemWithAttachmentsFixture(t, c.TestApp, core.Attrs{})

			res := c.Delete(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))

			expected := fmt.Sprintf("/sources/%d", mediaItem.SourceID)
			if redirected := res.RedirectedTo(t); redirected != expected {
				t.Errorf("expected redirect to %q, got %q", expected, redirected)
			}
		})

		t.Run("doesn't prevent re-download by default", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemWithAttachmentsFixture(t, c.TestApp, core.Attrs{})

			c.Delete(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))

			reloaded, _ := c.App.MediaGetMediaItem(c.Ctx, mediaItem.ID)
			if reloaded.PreventDownload {
				t.Errorf("expected prevent_download to be false by default")
			}
		})

		t.Run("can optionally prevent re-download", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemWithAttachmentsFixture(t, c.TestApp, core.Attrs{})

			c.Delete(fmt.Sprintf("/sources/%d/media/%d?prevent_download=true", mediaItem.SourceID, mediaItem.ID))

			reloaded, _ := c.App.MediaGetMediaItem(c.Ctx, mediaItem.ID)
			if !reloaded.PreventDownload {
				t.Errorf("expected prevent_download to be true")
			}
		})
	})
}

func TestMediaItemController_ForceDownload(t *testing.T) {
	t.Run("force_download", func(t *testing.T) {
		t.Run("enqueues download task", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{})

			jobs := c.Oban.Enqueued(t, obanlite.Match{})
			if len(jobs) != 0 {
				t.Errorf("expected no jobs enqueued initially")
			}

			c.Post(fmt.Sprintf("/sources/%d/media/%d/force_download", mediaItem.SourceID, mediaItem.ID), "", core.Attrs{})

			jobs = c.Oban.Enqueued(t, obanlite.Match{})
			if len(jobs) != 1 {
				t.Errorf("expected 1 job enqueued, got %d", len(jobs))
			}
		})

		t.Run("redirects to the show page", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{})

			res := c.Post(fmt.Sprintf("/sources/%d/media/%d/force_download", mediaItem.SourceID, mediaItem.ID), "", core.Attrs{})

			expected := fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID)
			if redirected := res.RedirectedTo(t); redirected != expected {
				t.Errorf("expected redirect to %q, got %q", expected, redirected)
			}
		})
	})
}

func TestMediaItemController_Stream(t *testing.T) {
	t.Run("streaming media", func(t *testing.T) {
		t.Run("returns 404 if the media isn't found", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{})

			uuid := ""
			if mediaItem.UUID != nil {
				uuid = *mediaItem.UUID
			}

			res := c.Get(fmt.Sprintf("/media/%s/stream", uuid))

			if res.Status != 404 {
				t.Errorf("expected 404, got %d", res.Status)
			}
		})

		t.Run("sets the content length", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := coretest.MediaItemWithAttachmentsFixture(t, c.TestApp, core.Attrs{})

			uuid := ""
			if mediaItem.UUID != nil {
				uuid = *mediaItem.UUID
			}

			stat, _ := os.Stat(*mediaItem.MediaFilepath)
			expected := fmt.Sprint(stat.Size())

			res := c.Get(fmt.Sprintf("/media/%s/stream", uuid))
			res.HTML(t, 200)

			cl := res.Header.Get("Content-Length")
			if cl != expected {
				t.Errorf("expected content-length %q, got %q", expected, cl)
			}
		})
	})
}
