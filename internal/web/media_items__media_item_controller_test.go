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

// mediaPath is the media item's page below its source, plus suffix.
func mediaPath(m *store.MediaItem, suffix string) string {
	return fmt.Sprintf("/sources/%d/media/%d%s", m.SourceID, m.ID, suffix)
}

func wantRedirect(t *testing.T, res *webtest.Response, expected string) {
	t.Helper()
	if redirected := res.RedirectedTo(t); redirected != expected {
		t.Errorf("expected redirect to %q, got %q", expected, redirected)
	}
}

func wantHeader(t *testing.T, res *webtest.Response, name, want string) {
	t.Helper()
	if got := res.Header.Get(name); got != want {
		t.Errorf("expected %s %q, got %q", strings.ToLower(name), want, got)
	}
}

func TestMediaItemController_Show(t *testing.T) {
	for _, c := range []struct {
		name string
		item store.MediaItemParams
		full bool // with attachments
	}{
		{"renders the page", store.MediaItemParams{}, false},
		{"renders the page when the media item has no description", store.MediaItemParams{Clear: store.ClearDescription}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			client := webtest.New(t)
			fixture := apptest.MediaItemFixture
			if c.full {
				fixture = apptest.MediaItemWithAttachmentsFixture
			}
			mediaItem := fixture(t, client.TestApp, c.item)

			html := client.Get(mediaPath(mediaItem, "")).HTML(t, 200)

			title := store.Deref(mediaItem.Title)
			if !strings.Contains(html, title) {
				t.Errorf("expected to find media title %q in response", title)
			}
		})
	}
}

func TestMediaItemController_Edit(t *testing.T) {
	t.Run("renders form for editing chosen media_item", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := createMediaItem(t, c)

		html := c.Get(mediaPath(mediaItem, "/edit")).HTML(t, 200)

		if !strings.Contains(html, "Editing") {
			t.Errorf("expected to find 'Editing' in response")
		}
	})
}

func TestMediaItemController_Update(t *testing.T) {
	t.Run("redirects when data is valid", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := createMediaItem(t, c)

		res := c.Patch(mediaPath(mediaItem, ""), "media_item", map[string]any{"title": "New Title"})

		wantRedirect(t, res, mediaPath(mediaItem, ""))
		if html := c.Get(mediaPath(mediaItem, "")).HTML(t, 200); !strings.Contains(html, "New Title") {
			t.Errorf("expected to find 'New Title' in response")
		}
	})

	t.Run("renders errors when data is invalid", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := createMediaItem(t, c)

		html := c.Patch(mediaPath(mediaItem, ""), "media_item", map[string]any{"title": nil}).HTML(t, 200)

		if !strings.Contains(html, "Editing") {
			t.Errorf("expected to find 'Editing' in response")
		}
	})
}

func TestMediaItemController_Delete(t *testing.T) {
	newFixture := func(t *testing.T) (*webtest.Client, *store.MediaItem) {
		c := webtest.New(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{})
		c.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })
		return c, mediaItem
	}

	t.Run("deletes the files but not the media item, redirects and doesn't prevent re-download by default", func(t *testing.T) {
		c, mediaItem := newFixture(t)
		filepath := store.Deref(mediaItem.MediaFilepath)

		res := c.Delete(mediaPath(mediaItem, ""))

		wantRedirect(t, res, fmt.Sprintf("/sources/%d", mediaItem.SourceID))
		reloaded, err := c.App.GetMediaItem(c.Ctx, mediaItem.ID)
		if err != nil || reloaded == nil {
			t.Fatalf("expected media item to still exist")
		}
		if reloaded.PreventDownload {
			t.Errorf("expected prevent_download to be false by default")
		}
		if filepath != "" {
			if _, err := os.Stat(filepath); err == nil {
				t.Errorf("expected media file to be deleted")
			}
		}
	})

	t.Run("can optionally prevent re-download", func(t *testing.T) {
		c, mediaItem := newFixture(t)

		c.Delete(mediaPath(mediaItem, "?prevent_download=true"))

		reloaded, err := c.App.GetMediaItem(c.Ctx, mediaItem.ID)
		must(t, err)
		if !reloaded.PreventDownload {
			t.Errorf("expected prevent_download to be true")
		}
	})
}

func TestMediaItemController_ForceDownload(t *testing.T) {
	forceDownload := func(c *webtest.Client, m *store.MediaItem) *webtest.Response {
		return c.Post(mediaPath(m, "/force_download"), "", map[string]any{})
	}
	wantJobs := func(t *testing.T, c *webtest.Client, args map[string]any, n int) {
		t.Helper()
		if jobs := c.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: args}); len(jobs) != n {
			t.Errorf("expected %d jobs enqueued, got %d", n, len(jobs))
		}
	}

	t.Run("enqueues download task", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := createMediaItem(t, c)
		wantJobs(t, c, nil, 0)

		forceDownload(c, mediaItem)

		wantJobs(t, c, nil, 1)
	})

	t.Run("doesn't freak out if the task is a duplicate", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := createMediaItem(t, c)
		c.App.MediaDownloadWorkerKickoffWithTask(c.Ctx, mediaItem, map[string]any{"force": true}, nil)

		forceDownload(c, mediaItem)

		wantJobs(t, c, nil, 1)
	})

	t.Run("forces a download even if one wouldn't normally run", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := apptest.MediaItemFixture(t, c.TestApp, store.MediaItemParams{Clear: store.ClearMediaFilepath})

		forceDownload(c, mediaItem)

		wantJobs(t, c, map[string]any{"id": mediaItem.ID, "force": true}, 1)
	})

	t.Run("redirects to the show page", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := createMediaItem(t, c)

		wantRedirect(t, forceDownload(c, mediaItem), mediaPath(mediaItem, ""))
	})
}

// streamMedia requests the media item's stream, with a Range header if rng is set.
func streamMedia(c *webtest.Client, m *store.MediaItem, rng string) *webtest.Response {
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/media/%s/stream", store.Deref(m.UUID)), nil)
	if rng != "" {
		req.Header.Set("Range", rng)
	}
	return c.Do(req)
}

func TestMediaItemController_Stream(t *testing.T) {
	t.Run("returns 404 if the media isn't found", func(t *testing.T) {
		c := webtest.New(t)

		if res := streamMedia(c, createMediaItem(t, c), ""); res.Status != 404 {
			t.Errorf("expected 404, got %d", res.Status)
		}
	})

	t.Run("automatically sets the content type and the content length", func(t *testing.T) {
		c := webtest.New(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{})
		stat, err := os.Stat(*mediaItem.MediaFilepath)
		must(t, err)

		res := streamMedia(c, mediaItem, "")

		wantHeader(t, res, "Content-Type", "video/mp4; charset=utf-8")
		wantHeader(t, res, "Content-Length", fmt.Sprint(stat.Size()))
	})
}

func TestMediaItemController_StreamRange(t *testing.T) {
	newFixture := func(t *testing.T) (*webtest.Client, *store.MediaItem, string) {
		c := webtest.New(t)
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, c.TestApp, store.MediaItemParams{})
		contents, err := os.ReadFile(*mediaItem.MediaFilepath)
		must(t, err)
		return c, mediaItem, string(contents)
	}
	wantDisposition := func(t *testing.T, res *webtest.Response, m *store.MediaItem) {
		t.Helper()
		wantHeader(t, res, "Content-Disposition", fmt.Sprintf("inline; filename=%q", *m.Title))
	}

	t.Run("when the range is valid", func(t *testing.T) {
		t.Run("sets the correct status and headers", func(t *testing.T) {
			c, mediaItem, contents := newFixture(t)

			res := streamMedia(c, mediaItem, "bytes=0-100")

			if res.Status != 206 {
				t.Errorf("expected status 206, got %d", res.Status)
			}
			wantHeader(t, res, "Content-Range", fmt.Sprintf("bytes 0-100/%d", len(contents)))
			wantHeader(t, res, "Content-Length", "101")
			wantDisposition(t, res, mediaItem)
		})

		for _, c := range []struct {
			name, rng  string
			from, to   int // expected slice of the file; to < 0 means the end
			fullLength bool
		}{
			{"streams the specified range", "bytes=0-100", 0, 101, false},
			{"supports range offsets", "bytes=100-200", 100, 201, false},
			{"returns as expected if the requested range is larger than the file", "bytes=0-99999999", 0, -1, true},
			{"supports endless ranges", "bytes=0-", 0, -1, false},
			{"supports endless ranges with offsets", "bytes=100-", 100, -1, false},
		} {
			t.Run(c.name, func(t *testing.T) {
				client, mediaItem, contents := newFixture(t)
				to := c.to
				if to < 0 {
					to = len(contents)
				}

				res := streamMedia(client, mediaItem, c.rng)

				if res.Body != contents[c.from:to] {
					t.Errorf("body did not match expected slice [%d:%d] (got %d bytes)", c.from, to, len(res.Body))
				}
				if c.fullLength {
					wantHeader(t, res, "Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(contents)-1, len(contents)))
					wantHeader(t, res, "Content-Length", fmt.Sprint(len(contents)))
				}
			})
		}
	})

	t.Run("when the range is invalid or not present", func(t *testing.T) {
		t.Run("sets the correct status and headers", func(t *testing.T) {
			c, mediaItem, contents := newFixture(t)

			res := streamMedia(c, mediaItem, "")

			if res.Status != 200 {
				t.Errorf("expected status 200, got %d", res.Status)
			}
			wantHeader(t, res, "Content-Length", fmt.Sprint(len(contents)))
			wantHeader(t, res, "Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(contents)-1, len(contents)))
			wantDisposition(t, res, mediaItem)
		})

		for _, c := range []struct{ name, rng string }{
			{"streams the entire file", ""},
			{"doesn't blow up if the range header is invalid", "bytes=-"},
		} {
			t.Run(c.name, func(t *testing.T) {
				client, mediaItem, contents := newFixture(t)

				res := streamMedia(client, mediaItem, c.rng)

				if res.Status != 200 {
					t.Errorf("expected status 200, got %d", res.Status)
				}
				if res.Body != contents {
					t.Errorf("expected full file contents")
				}
			})
		}
	})
}
