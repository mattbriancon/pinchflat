package app_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// downloadWorkerApp returns an app whose user scripts and HTTP client are
// stubbed out; callers install the yt-dlp mock they need.
func downloadWorkerApp(t *testing.T) *apptest.TestApp {
	ta := apptest.NewApp(t)
	ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })
	ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) { return "", nil })
	return ta
}

// performDownload runs the media download worker for a media item; extra are
// job args beyond the id.
func performDownload(ta *apptest.TestApp, id int64, extra map[string]any) error {
	args := map[string]any{"id": id}
	for k, v := range extra {
		args[k] = v
	}
	return ta.Oban.PerformJob(ta.Ctx, app.MediaDownloadWorkerName, args)
}

func TestMediaDownloadWorker_KickoffWithTask(t *testing.T) {
	t.Parallel()

	newItem := func(t *testing.T) (*apptest.TestApp, *store.MediaItem) {
		ta := apptest.NewApp(t)
		return ta, apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
	}

	t.Run("starts the worker", func(t *testing.T) {
		ta, mediaItem := newItem(t)

		if n := enqueuedDownloads(t, ta); n != 0 {
			t.Errorf("expected 0 enqueued jobs, got %d", n)
		}

		_, err := ta.MediaDownloadWorkerKickoffWithTask(ta.Ctx, mediaItem, map[string]any{}, nil)
		must(t, err)

		if n := enqueuedDownloads(t, ta); n != 1 {
			t.Errorf("expected 1 enqueued job, got %d", n)
		}
	})

	t.Run("attaches a task", func(t *testing.T) {
		ta, mediaItem := newItem(t)

		task, err := ta.MediaDownloadWorkerKickoffWithTask(ta.Ctx, mediaItem, map[string]any{}, nil)
		must(t, err)

		if task.MediaItemID == nil || *task.MediaItemID != mediaItem.ID {
			t.Errorf("expected task.MediaItemID %d, got %v", mediaItem.ID, task.MediaItemID)
		}
	})

	for _, c := range []struct {
		name         string
		args         map[string]any
		priority     *int
		wantPriority int
	}{
		{"can be called with additional job arguments", map[string]any{"force": true}, nil, 5},
		{"has a priority of 5 by default", map[string]any{}, nil, 5},
		{"priority can be set", map[string]any{}, store.Ptr(0), 0},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta, mediaItem := newItem(t)

			_, err := ta.MediaDownloadWorkerKickoffWithTask(ta.Ctx, mediaItem, c.args, c.priority)
			must(t, err)

			args := map[string]any{"id": mediaItem.ID}
			for k, v := range c.args {
				args[k] = v
			}
			jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.MediaDownloadWorkerName, Args: args})
			if len(jobs) != 1 {
				t.Fatalf("expected 1 job, got %d", len(jobs))
			}
			if jobs[0].Priority != c.wantPriority {
				t.Errorf("expected priority %d, got %v", c.wantPriority, jobs[0].Priority)
			}
		})
	}
}

func TestMediaDownloadWorker_Perform(t *testing.T) {
	t.Parallel()

	cleared := store.MediaItemParams{Clear: store.ClearMediaFilepath}
	downloaded := store.MediaItemParams{MediaFilepath: store.Ptr("foo.mp4")}
	quality := map[string]any{"quality_upgrade?": true}
	force := map[string]any{"force": true}
	noDownload := store.SourceParams{DownloadMedia: store.Ptr(false)}
	prevented := func(p store.MediaItemParams) store.MediaItemParams {
		p.PreventDownload = store.Ptr(true)
		return p
	}

	// Each case performs a job for one item and checks the item afterwards.
	for _, c := range []struct {
		name    string
		item    store.MediaItemParams
		args    map[string]any
		initial func(*store.MediaItem) bool // must hold before the job, if set
		ok      func(*store.MediaItem) bool // must hold on the reloaded item
	}{
		{"saves attributes to the media_item", cleared, nil, func(m *store.MediaItem) bool { return m.MediaFilepath == nil },
			func(m *store.MediaItem) bool { return m.MediaFilepath != nil }},
		{"saves the metadata to the media_item", cleared, nil, func(m *store.MediaItem) bool { return m.Metadata == nil },
			func(m *store.MediaItem) bool { return m.Metadata != nil }},
		{"does not set redownloaded_at by default", cleared, nil, nil,
			func(m *store.MediaItem) bool { return m.MediaRedownloadedAt == nil }},
		{"sets redownloaded_at on the media_item", cleared, quality, nil,
			func(m *store.MediaItem) bool { return m.MediaRedownloadedAt != nil }},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := downloadWorkerApp(t)
			ta.YtDlpMock.Run.Stub(downloadMock(nil))
			mediaItem := apptest.MediaItemFixture(t, ta, c.item)
			if c.initial != nil && !c.initial(mediaItem) {
				t.Errorf("initial state not as expected: %+v", mediaItem)
			}

			_ = performDownload(ta, mediaItem.ID, c.args)

			updated, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID)
			must(t, err)
			updated, err = ta.App.PreloadMediaItemMetadata(ta.Ctx, updated)
			must(t, err)
			if !c.ok(updated) {
				t.Errorf("unexpected media item after download: %+v", updated)
			}
		})
	}

	t.Run("won't double-schedule downloading jobs", func(t *testing.T) {
		ta := downloadWorkerApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, cleared)
		spec := obanlite.JobSpec{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}}

		ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), spec)
		ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), spec)

		if n := enqueuedDownloads(t, ta); n != 1 {
			t.Errorf("expected 1 job due to unique constraint, got %d", n)
		}
	})

	// A failing download either errors the job (so it is retried) or, when a
	// retry couldn't help, is swallowed.
	for _, c := range []struct {
		name         string
		msg          string
		uniqueJob    bool
		args         map[string]any
		wantRetryErr bool
	}{
		{"sets the job to retryable if the download fails", "error", true, nil, true},
		{"sets the job to retryable if the download failed and was retried", "Unable to communicate with SponsorBlock", true, nil, true},
		{"ensures error are returned in a 2-item tuple", "error", false, nil, true},
		{"does not set the job to retryable if retrying wouldn't fix the issue", "Something something Video unavailable something something", false, quality, false},
		{"does not set the job to retryable if youtube thinks you're a bot", "Sign in to confirm you're not a bot", false, quality, false},
		{"does not set the job to retryable you aren't a member", "This video is available to this channel's members on level: foo", false, quality, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := downloadWorkerApp(t)
			ta.YtDlpMock.Run.ExpectN(2, downloadMock(dlActions{"download": retErr(fmt.Errorf("%s", c.msg))}))
			mediaItem := apptest.MediaItemFixture(t, ta, cleared)
			if c.uniqueJob {
				_, _, _ = ta.InsertUniqueJob(ta.Ctx, obanlite.JobSpec{Worker: app.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})
			}

			err := performDownload(ta, mediaItem.ID, c.args)

			if c.wantRetryErr && err == nil {
				t.Error("expected an error so the job is retried")
			}
			if !c.wantRetryErr && err != nil {
				t.Errorf("expected nil error for non-retryable case, got %v", err)
			}
		})
	}

	t.Run("saves the file's size to the database", func(t *testing.T) {
		ta := downloadWorkerApp(t)
		ta.YtDlpMock.Run.ExpectN(3, downloadMock(dlActions{"download": func(ytCall) (string, error) {
			metadata, _ := apptest.RenderParsedMetadata("media_metadata")
			filePath, _ := metadata["filepath"].(string)

			// Write test file
			if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
				return "", err
			}
			return db.EncodeJSON(metadata)
		}}))
		mediaItem := apptest.MediaItemFixture(t, ta, cleared)

		_ = performDownload(ta, mediaItem.ID, nil)

		updated, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID)
		must(t, err)
		if updated.MediaSizeBytes == nil || *updated.MediaSizeBytes <= 0 {
			t.Error("expected media_size_bytes to be > 0")
		}
	})

	t.Run("does not blow up if the record doesn't exist", func(t *testing.T) {
		ta := downloadWorkerApp(t)
		ta.YtDlpMock.Run.Stub(downloadMock(nil))

		must(t, performDownload(ta, 0, nil))
	})

	// The force_overwrites runner option depends on whether the job is a
	// fresh download, a forced one or a quality upgrade.
	for _, c := range []struct {
		name string
		item store.MediaItemParams
		args map[string]any
		want string // "force_overwrites" or "no_force_overwrites"
	}{
		{"sets the no_force_overwrites runner option", cleared, nil, "no_force_overwrites"},
		{"sets force_overwrites runner option when forced", cleared, force, "force_overwrites"},
		{"sets force_overwrites runner option when redownloading", downloaded, quality, "force_overwrites"},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := downloadWorkerApp(t)
			other := "force_overwrites"
			if c.want == other {
				other = "no_force_overwrites"
			}
			ta.YtDlpMock.Run.ExpectN(3, downloadMock(dlActions{"download": func(c2 ytCall) (string, error) {
				wantOpts(t, c2.opts, []string{c.want}, []string{other})
				return retMetadata(c2)
			}}))
			mediaItem := apptest.MediaItemFixture(t, ta, c.item)

			_ = performDownload(ta, mediaItem.ID, c.args)
		})
	}

	// Each case must not reach the download runner and must not error.
	for _, c := range []struct {
		name string
		src  store.SourceParams
		item store.MediaItemParams
		args map[string]any
	}{
		{"does not download if the source is set to not download", noDownload, cleared, nil},
		{"does not download if the media item is set to not download", store.SourceParams{}, prevented(cleared), nil},
		{"does not download if the media item isn't pending download", store.SourceParams{}, downloaded, nil},
		{"doesn't redownload if the source is set to not download", noDownload, downloaded, quality},
		{"doesn't redownload if the media item is set to not download", store.SourceParams{}, prevented(downloaded), quality},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := downloadWorkerApp(t)
			ta.YtDlpMock.Run.Stub(downloadMock(dlActions{"download": func(ytCall) (string, error) {
				t.Error("download should not be called")
				return "{}", nil
			}}))
			c.item.SourceID = store.Ptr(apptest.SourceFixture(t, ta, c.src).ID)
			mediaItem := apptest.MediaItemFixture(t, ta, c.item)

			_ = performDownload(ta, mediaItem.ID, c.args)
		})
	}
}

func TestMediaDownloadWorker_Perform_WhenTestingNonDownloadableMedia(t *testing.T) {
	t.Run("does not retry the job if the media is currently not downloadable", func(t *testing.T) {
		ta := downloadWorkerApp(t)
		ta.YtDlpMock.Run.Expect(downloadMock(dlActions{"get_downloadable_status": ret(`{"live_status": "is_live"}`)}))
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})

		// Download should be skipped for non-downloadable media, so no error expected
		must(t, performDownload(ta, mediaItem.ID, nil))
	})
}

func TestMediaDownloadWorker_Perform_WhenTestingForcedDownloads(t *testing.T) {
	// Forced and quality-upgrade jobs ignore prevent_download and whether the
	// item is pending, and must succeed.
	for _, c := range []struct {
		name string
		src  store.SourceParams
		item store.MediaItemParams
		args map[string]any
	}{
		{"ignores 'prevent_download' if forced", store.SourceParams{DownloadMedia: store.Ptr(false)}, store.MediaItemParams{PreventDownload: store.Ptr(true), Clear: store.ClearMediaFilepath}, map[string]any{"force": true}},
		{"ignores whether the media item is pending when forced", store.SourceParams{}, store.MediaItemParams{MediaFilepath: store.Ptr("foo.mp4")}, map[string]any{"force": true}},
		{"ignores whether the media item is pending when re-downloaded", store.SourceParams{}, store.MediaItemParams{MediaFilepath: store.Ptr("foo.mp4")}, map[string]any{"quality_upgrade?": true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := downloadWorkerApp(t)
			ta.YtDlpMock.Run.Stub(downloadMock(nil))
			c.item.SourceID = store.Ptr(apptest.SourceFixture(t, ta, c.src).ID)
			mediaItem := apptest.MediaItemFixture(t, ta, c.item)

			must(t, performDownload(ta, mediaItem.ID, c.args))
		})
	}

	t.Run("deletes old files if the media item has been updated", func(t *testing.T) {
		ta := downloadWorkerApp(t)
		ta.YtDlpMock.Run.ExpectN(3, downloadMock(dlActions{"download": func(ytCall) (string, error) {
			metadata, _ := apptest.RenderParsedMetadata("media_metadata")
			metadata["filepath"] = apptest.MediaFilePathFixture() // Use old path
			return db.EncodeJSON(metadata)
		}}))
		oldMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})
		oldFilepath := *oldMediaItem.MediaFilepath

		_ = performDownload(ta, oldMediaItem.ID, map[string]any{"force": true})

		updated, err := ta.GetMediaItem(ta.Ctx, oldMediaItem.ID)
		must(t, err)
		if updated.MediaFilepath == nil || *updated.MediaFilepath == oldFilepath {
			t.Error("media_filepath should be updated")
		}
		if _, err := os.Stat(oldFilepath); err == nil {
			t.Error("old file should be deleted")
		}
	})
}

func TestMediaDownloadWorker_Perform_WhenTestingUserScriptCallbacks(t *testing.T) {
	t.Run("calls the media_pre_download and media_downloaded user script runners", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Stub(downloadMock(nil))
		items := map[string]*store.MediaItem{}
		ta.UserScriptMock.Run.ExpectN(2, func(event string, data any) error {
			if item, ok := data.(*store.MediaItem); ok {
				items[event] = item
			}
			return nil
		})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})

		_ = performDownload(ta, mediaItem.ID, nil)

		for _, event := range []string{"media_pre_download", "media_downloaded"} {
			if items[event] == nil || items[event].ID != mediaItem.ID {
				t.Errorf("expected %s to be called with the correct media item", event)
			}
		}
		updated, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID)
		must(t, err)
		if updated.MediaFilepath == nil {
			t.Error("expected media_filepath to be set")
		}
		if updated.PreventDownload {
			t.Error("expected prevent_download to be false")
		}
	})

	t.Run("does not download the media if the pre-download script returns an error", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ta.YtDlpMock.Run.Stub(downloadMock(dlActions{"download": func(ytCall) (string, error) {
			t.Error("download should not be called when pre-download script fails")
			return "{}", nil
		}}))
		ta.UserScriptMock.Run.Expect(func(event string, data any) error {
			if event == "media_pre_download" {
				return fmt.Errorf("exit code 1")
			}
			return nil
		})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})

		_ = performDownload(ta, mediaItem.ID, nil)

		updated, err := ta.GetMediaItem(ta.Ctx, mediaItem.ID)
		must(t, err)
		if !updated.PreventDownload {
			t.Error("expected prevent_download to be set to true")
		}
	})
}
