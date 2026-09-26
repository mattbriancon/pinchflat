package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestMediaDownloadWorker_KickoffWithTask(t *testing.T) {
	t.Run("starts the worker", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		enqueuedJobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(enqueuedJobs) != 0 {
			t.Errorf("expected 0 enqueued jobs, got %d", len(enqueuedJobs))
		}

		_, err := ta.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, core.Attrs{}, core.KW{})
		if err != nil {
			t.Fatalf("MediaDownloadWorkerKickoffWithTask failed: %v", err)
		}

		enqueuedJobs = ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(enqueuedJobs) != 1 {
			t.Errorf("expected 1 enqueued job, got %d", len(enqueuedJobs))
		}
	})

	t.Run("attaches a task", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		task, err := ta.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, core.Attrs{}, core.KW{})
		if err != nil {
			t.Fatalf("MediaDownloadWorkerKickoffWithTask failed: %v", err)
		}

		if task.MediaItemID == nil || *task.MediaItemID != mediaItem.ID {
			t.Errorf("expected task.MediaItemID %d, got %v", mediaItem.ID, task.MediaItemID)
		}
	})

	t.Run("can be called with additional job arguments", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})
		jobArgs := core.Attrs{"force": true}

		_, err := ta.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, jobArgs, core.KW{})
		if err != nil {
			t.Fatalf("MediaDownloadWorkerKickoffWithTask failed: %v", err)
		}

		ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID, "force": true}})
	})

	t.Run("has a priority of 5 by default", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_, err := ta.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, core.Attrs{}, core.KW{})
		if err != nil {
			t.Fatalf("MediaDownloadWorkerKickoffWithTask failed: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(jobs))
		}

		job := jobs[0]
		if job.Priority != 5 {
			t.Errorf("expected priority 5, got %v", job.Priority)
		}
	})

	t.Run("priority can be set", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})
		priority := 0

		_, err := ta.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, core.Attrs{}, core.KW{core.Opt("priority", priority)})
		if err != nil {
			t.Fatalf("MediaDownloadWorkerKickoffWithTask failed: %v", err)
		}

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}})
		if len(jobs) != 1 {
			t.Fatalf("expected 1 job, got %d", len(jobs))
		}

		job := jobs[0]
		if job.Priority != 0 {
			t.Errorf("expected priority 0, got %v", job.Priority)
		}
	})
}

func TestMediaDownloadWorker_Perform(t *testing.T) {
	setup := func(t *testing.T) *coretest.TestApp {
		ta := coretest.NewApp(t)

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		ta.HTTPMock.Get.Stub(func(url string, headers, opts core.KW) (string, error) {
			return "", nil
		})

		return ta
	}

	t.Run("does not blow up if the record doesn't exist", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": int64(0)})
		if err != nil {
			t.Fatalf("expected nil error for missing media item, got %v", err)
		}
	})

	t.Run("does not download if the source is set to not download", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		// Stub download to fail if called
		downloadCalled := false
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "download" {
				downloadCalled = true
			}
			return "{}", nil
		})

		source := coretest.SourceFixture(t, ta, core.Attrs{"download_media": false})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		if downloadCalled {
			t.Error("download should not be called")
		}
	})

	t.Run("does not download if the media item is set to not download", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		downloadCalled := false
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "download" {
				downloadCalled = true
			}
			return "{}", nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})
		_, _ = ta.MediaUpdateMediaItem(ctx, mediaItem, core.Attrs{"prevent_download": true})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		if downloadCalled {
			t.Error("download should not be called")
		}
	})

	t.Run("does not download if the media item isn't pending download", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		downloadCalled := false
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "download" {
				downloadCalled = true
			}
			return "{}", nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": "foo.mp4"})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		if downloadCalled {
			t.Error("download should not be called")
		}
	})
}

func TestMediaDownloadWorker_Perform_WhenTestingForcedDownloads(t *testing.T) {
	t.Run("ignores 'prevent_download' if forced", func(t *testing.T) {
	})

	t.Run("ignores whether the media item is pending when forced", func(t *testing.T) {
	})

	t.Run("sets force_overwrites runner option", func(t *testing.T) {
	})
}

func TestMediaDownloadWorker_Perform_WhenTestingRedownloads(t *testing.T) {
	t.Run("sets redownloaded_at on the media_item", func(t *testing.T) {
	})

	t.Run("ignores whether the media item is pending when re-downloaded", func(t *testing.T) {
	})

	t.Run("doesn't redownload if the source is set to not download", func(t *testing.T) {
	})

	t.Run("doesn't redownload if the media item is set to not download", func(t *testing.T) {
	})

	t.Run("sets force_overwrites runner option", func(t *testing.T) {
	})

	t.Run("deletes old files if the media item has been updated", func(t *testing.T) {
	})
}

func TestMediaDownloadWorker_Perform_WhenTestingUserScriptCallbacks(t *testing.T) {
	t.Run("calls the media_pre_download user script runner", func(t *testing.T) {
	})

	t.Run("does not download the media if the pre-download script returns an error", func(t *testing.T) {
	})

	t.Run("downloads media if the pre-download script is not present", func(t *testing.T) {
	})

	t.Run("calls the media_downloaded user script runner", func(t *testing.T) {
	})
}
