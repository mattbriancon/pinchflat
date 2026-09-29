package web_test

// Port of test/pinchflat_web/controllers/media_item_controller_test.exs

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func createMediaItem(t testing.TB, c *webtest.Client) *store.MediaItem {
	return apptest.MediaItemFixture(t, c.TestApp, store.MediaItemParams{})
}

func TestMediaItemController_Show(t *testing.T) {
	t.Run("show media", func(t *testing.T) {
		t.Run("renders the page", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := createMediaItem(t, c)

			res := c.Get(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))
			html := res.HTML(t, 200)

			title := ""
			if mediaItem.Title != nil {
				title = *mediaItem.Title
			}
			if !strings.Contains(html, title) {
				t.Errorf("expected to find media title %q in response", title)
			}
		})

		t.Run("renders the page when the media item has no description", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{Clear: store.ClearDescription})

			res := c.Get(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))
			html := res.HTML(t, 200)

			title := ""
			if mediaItem.Title != nil {
				title = *mediaItem.Title
			}
			if !strings.Contains(html, title) {
				t.Errorf("expected to find media title %q in response", title)
			}
		})
	})
}

func TestMediaItemController_Edit(t *testing.T) {
	t.Run("edit media", func(t *testing.T) {
		t.Run("renders form for editing chosen media_item", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := createMediaItem(t, c)

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
			mediaItem := createMediaItem(t, c)

			updateAttrs := map[string]any{"title": "New Title"}
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
			mediaItem := createMediaItem(t, c)

			res := c.Patch(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID), "media_item", map[string]any{"title": nil})
			html := res.HTML(t, 200)

			if !strings.Contains(html, "Editing") {
				t.Errorf("expected to find 'Editing' in response")
			}
		})
	})
}

func TestMediaItemController_Delete(t *testing.T) {
	t.Run("delete media", func(t *testing.T) {
		newFixture := func(t testing.TB, c *webtest.Client) *store.MediaItem {
			mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{})
			c.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })
			return mediaItem
		}

		t.Run("the media item not is deleted", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := newFixture(t, c)

			c.Delete(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))

			reloaded, err := c.App.GetMediaItem(c.Ctx, mediaItem.ID)
			if err != nil || reloaded == nil {
				t.Errorf("expected media item to still exist")
			}
		})

		t.Run("the files are deleted", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := newFixture(t, c)

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
			mediaItem := newFixture(t, c)

			res := c.Delete(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))

			expected := fmt.Sprintf("/sources/%d", mediaItem.SourceID)
			if redirected := res.RedirectedTo(t); redirected != expected {
				t.Errorf("expected redirect to %q, got %q", expected, redirected)
			}
		})

		t.Run("doesn't prevent re-download by default", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := newFixture(t, c)

			c.Delete(fmt.Sprintf("/sources/%d/media/%d", mediaItem.SourceID, mediaItem.ID))

			reloaded, _ := c.App.GetMediaItem(c.Ctx, mediaItem.ID)
			if reloaded.PreventDownload {
				t.Errorf("expected prevent_download to be false by default")
			}
		})

		t.Run("can optionally prevent re-download", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := newFixture(t, c)

			c.Delete(fmt.Sprintf("/sources/%d/media/%d?prevent_download=true", mediaItem.SourceID, mediaItem.ID))

			reloaded, _ := c.App.GetMediaItem(c.Ctx, mediaItem.ID)
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
			mediaItem := createMediaItem(t, c)

			if jobs := c.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName}); len(jobs) != 0 {
				t.Errorf("expected no jobs enqueued initially")
			}

			c.Post(fmt.Sprintf("/sources/%d/media/%d/force_download", mediaItem.SourceID, mediaItem.ID), "", map[string]any{})

			jobs := c.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
			if len(jobs) != 1 {
				t.Errorf("expected 1 job enqueued, got %d", len(jobs))
			}
		})

		t.Run("doesn't freak out if the task is a duplicate", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := createMediaItem(t, c)

			c.App.MediaDownloadWorkerKickoffWithTask(c.Ctx, mediaItem, map[string]any{"force": true}, nil)

			c.Post(fmt.Sprintf("/sources/%d/media/%d/force_download", mediaItem.SourceID, mediaItem.ID), "", map[string]any{})

			jobs := c.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName})
			if len(jobs) != 1 {
				t.Errorf("expected 1 job enqueued, got %d", len(jobs))
			}
		})

		t.Run("forces a download even if one wouldn't normally run", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := apptest.MediaItemFixture(t, c.TestApp, store.MediaItemParams{Clear: store.ClearMediaFilepath})

			c.Post(fmt.Sprintf("/sources/%d/media/%d/force_download", mediaItem.SourceID, mediaItem.ID), "", map[string]any{})

			jobs := c.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID, "force": true}})
			if len(jobs) != 1 {
				t.Errorf("expected 1 job enqueued, got %d", len(jobs))
			}
		})

		t.Run("redirects to the show page", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := createMediaItem(t, c)

			res := c.Post(fmt.Sprintf("/sources/%d/media/%d/force_download", mediaItem.SourceID, mediaItem.ID), "", map[string]any{})

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
			mediaItem := createMediaItem(t, c)

			uuid := ""
			if mediaItem.UUID != nil {
				uuid = *mediaItem.UUID
			}

			res := c.Get(fmt.Sprintf("/media/%s/stream", uuid))

			if res.Status != 404 {
				t.Errorf("expected 404, got %d", res.Status)
			}
		})

		t.Run("automatically sets the content type", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{})

			uuid := ""
			if mediaItem.UUID != nil {
				uuid = *mediaItem.UUID
			}

			res := c.Get(fmt.Sprintf("/media/%s/stream", uuid))

			if ct := res.Header.Get("Content-Type"); ct != "video/mp4; charset=utf-8" {
				t.Errorf("expected content-type %q, got %q", "video/mp4; charset=utf-8", ct)
			}
		})

		t.Run("sets the content length", func(t *testing.T) {
			c := webtest.New(t)
			mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{})

			uuid := ""
			if mediaItem.UUID != nil {
				uuid = *mediaItem.UUID
			}

			stat, err := os.Stat(*mediaItem.MediaFilepath)
			must(t, err)
			expected := fmt.Sprint(stat.Size())

			res := c.Get(fmt.Sprintf("/media/%s/stream", uuid))

			cl := res.Header.Get("Content-Length")
			if cl != expected {
				t.Errorf("expected content-length %q, got %q", expected, cl)
			}
		})
	})
}

func TestMediaItemController_StreamRangeValid(t *testing.T) {
	t.Run("streaming media when range is valid", func(t *testing.T) {
		newFixture := func(t testing.TB) (*webtest.Client, *store.MediaItem) {
			c := webtest.New(t)
			mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{})
			return c, mediaItem
		}

		t.Run("sets the correct status and headers", func(t *testing.T) {
			c, mediaItem := newFixture(t)

			stat, err := os.Stat(*mediaItem.MediaFilepath)
			must(t, err)
			filesize := stat.Size()

			uuid := *mediaItem.UUID
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/media/%s/stream", uuid), nil)
			req.Header.Set("Range", "bytes=0-100")
			res := c.Do(req)

			if res.Status != 206 {
				t.Errorf("expected status 206, got %d", res.Status)
			}
			expectedRange := fmt.Sprintf("bytes 0-100/%d", filesize)
			if got := res.Header.Get("Content-Range"); got != expectedRange {
				t.Errorf("expected content-range %q, got %q", expectedRange, got)
			}
			if got := res.Header.Get("Content-Length"); got != "101" {
				t.Errorf("expected content-length %q, got %q", "101", got)
			}
			expectedDisposition := fmt.Sprintf("inline; filename=%q", *mediaItem.Title)
			if got := res.Header.Get("Content-Disposition"); got != expectedDisposition {
				t.Errorf("expected content-disposition %q, got %q", expectedDisposition, got)
			}
		})

		t.Run("streams the specified range", func(t *testing.T) {
			c, mediaItem := newFixture(t)

			uuid := *mediaItem.UUID
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/media/%s/stream", uuid), nil)
			req.Header.Set("Range", "bytes=0-100")
			res := c.Do(req)

			if len(res.Body) != 101 {
				t.Errorf("expected 101 bytes, got %d", len(res.Body))
			}
		})

		t.Run("supports range offsets", func(t *testing.T) {
			c, mediaItem := newFixture(t)

			contents, err := os.ReadFile(*mediaItem.MediaFilepath)
			must(t, err)
			expected := string(contents[100:201])

			uuid := *mediaItem.UUID
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/media/%s/stream", uuid), nil)
			req.Header.Set("Range", "bytes=100-200")
			res := c.Do(req)

			if res.Body != expected {
				t.Errorf("body did not match expected range slice")
			}
		})

		t.Run("returns as expected if the requested range is larger than the file", func(t *testing.T) {
			c, mediaItem := newFixture(t)

			contents, err := os.ReadFile(*mediaItem.MediaFilepath)
			must(t, err)
			filesize := int64(len(contents))

			uuid := *mediaItem.UUID
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/media/%s/stream", uuid), nil)
			req.Header.Set("Range", fmt.Sprintf("bytes=0-%d", filesize*10))
			res := c.Do(req)

			if res.Body != string(contents) {
				t.Errorf("expected full file contents")
			}
			expectedRange := fmt.Sprintf("bytes 0-%d/%d", filesize-1, filesize)
			if got := res.Header.Get("Content-Range"); got != expectedRange {
				t.Errorf("expected content-range %q, got %q", expectedRange, got)
			}
			if got := res.Header.Get("Content-Length"); got != fmt.Sprint(filesize) {
				t.Errorf("expected content-length %q, got %q", fmt.Sprint(filesize), got)
			}
		})

		t.Run("supports endless ranges", func(t *testing.T) {
			c, mediaItem := newFixture(t)

			contents, err := os.ReadFile(*mediaItem.MediaFilepath)
			must(t, err)

			uuid := *mediaItem.UUID
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/media/%s/stream", uuid), nil)
			req.Header.Set("Range", "bytes=0-")
			res := c.Do(req)

			if res.Body != string(contents) {
				t.Errorf("expected full file contents")
			}
		})

		t.Run("supports endless ranges with offsets", func(t *testing.T) {
			c, mediaItem := newFixture(t)

			contents, err := os.ReadFile(*mediaItem.MediaFilepath)
			must(t, err)
			expected := string(contents[100:])

			uuid := *mediaItem.UUID
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/media/%s/stream", uuid), nil)
			req.Header.Set("Range", "bytes=100-")
			res := c.Do(req)

			if res.Body != expected {
				t.Errorf("body did not match expected offset slice")
			}
		})
	})
}

func TestMediaItemController_StreamRangeInvalid(t *testing.T) {
	t.Run("streaming media when range is invalid or not present", func(t *testing.T) {
		newFixture := func(t testing.TB) (*webtest.Client, *store.MediaItem) {
			c := webtest.New(t)
			mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{})
			return c, mediaItem
		}

		t.Run("sets the correct status and headers", func(t *testing.T) {
			c, mediaItem := newFixture(t)

			stat, err := os.Stat(*mediaItem.MediaFilepath)
			must(t, err)
			filesize := stat.Size()

			uuid := *mediaItem.UUID
			res := c.Get(fmt.Sprintf("/media/%s/stream", uuid))

			if res.Status != 200 {
				t.Errorf("expected status 200, got %d", res.Status)
			}
			if got := res.Header.Get("Content-Length"); got != fmt.Sprint(filesize) {
				t.Errorf("expected content-length %q, got %q", fmt.Sprint(filesize), got)
			}
			expectedRange := fmt.Sprintf("bytes 0-%d/%d", filesize-1, filesize)
			if got := res.Header.Get("Content-Range"); got != expectedRange {
				t.Errorf("expected content-range %q, got %q", expectedRange, got)
			}
			expectedDisposition := fmt.Sprintf("inline; filename=%q", *mediaItem.Title)
			if got := res.Header.Get("Content-Disposition"); got != expectedDisposition {
				t.Errorf("expected content-disposition %q, got %q", expectedDisposition, got)
			}
		})

		t.Run("streams the entire file", func(t *testing.T) {
			c, mediaItem := newFixture(t)

			contents, err := os.ReadFile(*mediaItem.MediaFilepath)
			must(t, err)

			uuid := *mediaItem.UUID
			res := c.Get(fmt.Sprintf("/media/%s/stream", uuid))

			if res.Body != string(contents) {
				t.Errorf("expected full file contents")
			}
		})

		t.Run("doesn't blow up if the range header is invalid", func(t *testing.T) {
			c, mediaItem := newFixture(t)

			contents, err := os.ReadFile(*mediaItem.MediaFilepath)
			must(t, err)

			uuid := *mediaItem.UUID
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/media/%s/stream", uuid), nil)
			req.Header.Set("Range", "bytes=-")
			res := c.Do(req)

			if res.Status != 200 {
				t.Errorf("expected status 200, got %d", res.Status)
			}
			if res.Body != string(contents) {
				t.Errorf("expected full file contents")
			}
		})
	})
}
