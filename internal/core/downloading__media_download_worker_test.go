package core_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestMediaDownloadWorker_KickoffWithTask(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	setup := func(t *testing.T) *coretest.TestApp {
		ta := coretest.NewApp(t)

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				metadata, _ := coretest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
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

	t.Run("saves attributes to the media_item", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		if mediaItem.MediaFilepath != nil {
			t.Errorf("expected media_filepath to be nil, got %v", *mediaItem.MediaFilepath)
		}

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})
		updatedMediaItem, _ := ta.MediaGetMediaItem(ctx, mediaItem.ID)

		if updatedMediaItem.MediaFilepath == nil {
			t.Error("expected media_filepath to not be nil after download")
		}
	})

	t.Run("saves the metadata to the media_item", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		if mediaItem.Metadata != nil {
			t.Errorf("expected metadata to be nil, got %v", mediaItem.Metadata)
		}

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})
		updatedMediaItem, _ := ta.MediaGetMediaItem(ctx, mediaItem.ID)
		updatedMediaItem, _ = ta.App.PreloadMediaItemMetadata(ctx, updatedMediaItem)

		if updatedMediaItem.Metadata == nil {
			t.Error("expected metadata to not be nil after download")
		}
	})

	t.Run("won't double-schedule downloading jobs", func(t *testing.T) {
		ta := setup(t)

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		spec1 := obanlite.JobSpec{Worker: core.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}}
		spec2 := obanlite.JobSpec{Worker: core.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}}

		ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), spec1)
		ta.Oban.Insert(ta.Ctx, ta.App.Q(ta.Ctx), spec2)

		jobs := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.MediaDownloadWorkerName})
		if len(jobs) != 1 {
			t.Errorf("expected 1 job due to unique constraint, got %d", len(jobs))
		}
	})

	t.Run("sets the job to retryable if the download fails", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				return "", fmt.Errorf("error")
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})
		spec := obanlite.JobSpec{Worker: core.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}}
		_, _, _ = ta.InsertUniqueJob(ctx, spec)

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		if err == nil {
			t.Error("expected error for download failure")
		}
	})

	t.Run("sets the job to retryable if the download failed and was retried", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				return "", fmt.Errorf("Unable to communicate with SponsorBlock")
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})
		spec := obanlite.JobSpec{Worker: core.MediaDownloadWorkerName, Args: map[string]any{"id": mediaItem.ID}}
		_, _, _ = ta.InsertUniqueJob(ctx, spec)

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		// If it's a retryable error, it should return an error
		if err == nil {
			t.Error("expected error for retryable case")
		}
	})

	t.Run("does not set the job to retryable if retrying wouldn't fix the issue", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				return "", fmt.Errorf("Something something Video unavailable something something")
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "quality_upgrade?": true})

		if err != nil {
			t.Errorf("expected nil error for non-retryable case, got %v", err)
		}
	})

	t.Run("does not set the job to retryable if youtube thinks you're a bot", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				return "", fmt.Errorf("Sign in to confirm you're not a bot")
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "quality_upgrade?": true})

		if err != nil {
			t.Errorf("expected nil error for bot check case, got %v", err)
		}
	})

	t.Run("does not set the job to retryable you aren't a member", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				return "", fmt.Errorf("This video is available to this channel's members on level: foo")
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "quality_upgrade?": true})

		if err != nil {
			t.Errorf("expected nil error for members-only case, got %v", err)
		}
	})

	t.Run("ensures error are returned in a 2-item tuple", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				return "", fmt.Errorf("error")
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		if err == nil {
			t.Error("expected error for download_failed case")
		}
	})

	t.Run("saves the file's size to the database", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		callCount := 0
		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			defer func() { callCount++ }()

			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				metadata, _ := coretest.RenderParsedMetadata("media_metadata")
				filePath, _ := metadata["filepath"].(string)

				// Write test file
				if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
					return "", err
				}
				if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
					return "", err
				}

				jsonStr, _ := db.EncodeJSON(metadata)
				return jsonStr, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
			return "{}", nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})
		updatedMediaItem, _ := ta.MediaGetMediaItem(ctx, mediaItem.ID)

		if updatedMediaItem.MediaSizeBytes == nil || *updatedMediaItem.MediaSizeBytes <= 0 {
			t.Error("expected media_size_bytes to be > 0")
		}
	})

	t.Run("does not set redownloaded_at by default", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})
		updatedMediaItem, _ := ta.MediaGetMediaItem(ctx, mediaItem.ID)

		if updatedMediaItem.MediaRedownloadedAt != nil {
			t.Errorf("expected media_redownloaded_at to be nil, got %v", updatedMediaItem.MediaRedownloadedAt)
		}
	})

	t.Run("does not blow up if the record doesn't exist", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": int64(0)})
		if err != nil {
			t.Fatalf("expected nil error for missing media item, got %v", err)
		}
	})

	t.Run("sets the no_force_overwrites runner option", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		noForceOverwritesCalled := false
		forceOverwritesCalled := false

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				// Check that no_force_overwrites is in opts and force_overwrites is not
				for _, opt := range opts {
					if opt.Key == "no_force_overwrites" {
						noForceOverwritesCalled = true
					}
					if opt.Key == "force_overwrites" {
						forceOverwritesCalled = true
					}
				}
				metadata, _ := coretest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		if !noForceOverwritesCalled {
			t.Error("expected no_force_overwrites to be called")
		}
		if forceOverwritesCalled {
			t.Error("expected force_overwrites to not be called")
		}
	})

	t.Run("does not download if the source is set to not download", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

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

func TestMediaDownloadWorker_Perform_WhenTestingNonDownloadableMedia(t *testing.T) {
	t.Run("does not retry the job if the media is currently not downloadable", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return `{"live_status": "is_live"}`, nil
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		if err == nil {
			// Download should be skipped for non-downloadable media, so no error expected
		} else {
			t.Errorf("expected no error for non-downloadable media, got %v", err)
		}
	})
}

func TestMediaDownloadWorker_Perform_WhenTestingForcedDownloads(t *testing.T) {
	setup := func(t *testing.T) *coretest.TestApp {
		ta := coretest.NewApp(t)

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				metadata, _ := coretest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
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

	t.Run("ignores 'prevent_download' if forced", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		source := coretest.SourceFixture(t, ta, core.Attrs{"download_media": false})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": nil})

		_, _ = ta.MediaUpdateMediaItem(ctx, mediaItem, core.Attrs{"prevent_download": true})

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "force": true})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("ignores whether the media item is pending when forced", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": "foo.mp4"})

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "force": true})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("sets force_overwrites runner option", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		forceOverwritesCalled := false
		noForceOverwritesCalled := false

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				for _, opt := range opts {
					if opt.Key == "force_overwrites" {
						forceOverwritesCalled = true
					}
					if opt.Key == "no_force_overwrites" {
						noForceOverwritesCalled = true
					}
				}
				metadata, _ := coretest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "force": true})

		if !forceOverwritesCalled {
			t.Error("expected force_overwrites to be called")
		}
		if noForceOverwritesCalled {
			t.Error("expected no_force_overwrites to not be called")
		}
	})
}

func TestMediaDownloadWorker_Perform_WhenTestingRedownloads(t *testing.T) {
	setup := func(t *testing.T) *coretest.TestApp {
		ta := coretest.NewApp(t)

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				metadata, _ := coretest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
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

	t.Run("sets redownloaded_at on the media_item", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "quality_upgrade?": true})
		updatedMediaItem, _ := ta.MediaGetMediaItem(ctx, mediaItem.ID)

		if updatedMediaItem.MediaRedownloadedAt == nil {
			t.Error("expected media_redownloaded_at to be set")
		}
	})

	t.Run("ignores whether the media item is pending when re-downloaded", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": "foo.mp4"})

		err := ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "quality_upgrade?": true})

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("doesn't redownload if the source is set to not download", func(t *testing.T) {
		ta := setup(t)
		ctx := ta.Ctx

		downloadCalled := false
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "download" {
				downloadCalled = true
			}
			return "{}", nil
		})

		source := coretest.SourceFixture(t, ta, core.Attrs{"download_media": false})
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": "foo.mp4"})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "quality_upgrade?": true})

		if downloadCalled {
			t.Error("download should not be called")
		}
	})

	t.Run("doesn't redownload if the media item is set to not download", func(t *testing.T) {
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
		_, _ = ta.MediaUpdateMediaItem(ctx, mediaItem, core.Attrs{"prevent_download": true})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "quality_upgrade?": true})

		if downloadCalled {
			t.Error("download should not be called")
		}
	})

	t.Run("sets force_overwrites runner option", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		forceOverwritesCalled := false
		noForceOverwritesCalled := false

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				for _, opt := range opts {
					if opt.Key == "force_overwrites" {
						forceOverwritesCalled = true
					}
					if opt.Key == "no_force_overwrites" {
						noForceOverwritesCalled = true
					}
				}
				metadata, _ := coretest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": "foo.mp4"})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID, "quality_upgrade?": true})

		if !forceOverwritesCalled {
			t.Error("expected force_overwrites to be called")
		}
		if noForceOverwritesCalled {
			t.Error("expected no_force_overwrites to not be called")
		}
	})

	t.Run("deletes old files if the media item has been updated", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				metadata, _ := coretest.RenderParsedMetadata("media_metadata")
				metadata["filepath"] = coretest.MediaFilePathFixture() // Use old path
				jsonStr, _ := db.EncodeJSON(metadata)
				return jsonStr, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		oldMediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})
		oldFilepath := *oldMediaItem.MediaFilepath

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": oldMediaItem.ID, "force": true})
		updatedMediaItem, _ := ta.MediaGetMediaItem(ctx, oldMediaItem.ID)

		if updatedMediaItem.MediaFilepath == nil || *updatedMediaItem.MediaFilepath == oldFilepath {
			t.Error("media_filepath should be updated")
		}

		if _, err := os.Stat(oldFilepath); err == nil {
			t.Error("old file should be deleted")
		}
	})
}

func TestMediaDownloadWorker_Perform_WhenTestingUserScriptCallbacks(t *testing.T) {
	t.Run("calls the media_pre_download user script runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		preDownloadCalled := false
		preDownloadMediaItem := (*core.MediaItem)(nil)

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				metadata, _ := coretest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.ExpectN(2, func(event string, data any) error {
			if event == "media_pre_download" {
				preDownloadCalled = true
				if item, ok := data.(*core.MediaItem); ok {
					preDownloadMediaItem = item
				}
			}
			if event == "media_downloaded" {
				// Just return nil for post-download
			}
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		if !preDownloadCalled {
			t.Error("expected media_pre_download to be called")
		}
		if preDownloadMediaItem == nil || preDownloadMediaItem.ID != mediaItem.ID {
			t.Error("expected pre-download script to receive the correct media item")
		}
	})

	t.Run("does not download the media if the pre-download script returns an error", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		downloadCalled := false

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "download" {
				downloadCalled = true
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.Expect(func(event string, data any) error {
			if event == "media_pre_download" {
				return fmt.Errorf("exit code 1")
			}
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		updatedMediaItem, _ := ta.MediaGetMediaItem(ctx, mediaItem.ID)

		if downloadCalled {
			t.Error("download should not be called when pre-download script fails")
		}
		if !updatedMediaItem.PreventDownload {
			t.Error("expected prevent_download to be set to true")
		}
	})

	t.Run("downloads media if the pre-download script is not present", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		downloadCalled := false

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				downloadCalled = true
				metadata, _ := coretest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
			return "{}", nil
		})

		callCount := 0
		ta.UserScriptMock.Run.ExpectN(2, func(event string, data any) error {
			defer func() { callCount++ }()
			if event == "media_pre_download" || event == "media_downloaded" {
				// Return nil (no error) to simulate script not being present
				return nil
			}
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		updatedMediaItem, _ := ta.MediaGetMediaItem(ctx, mediaItem.ID)

		if !downloadCalled {
			t.Error("download should be called")
		}
		if updatedMediaItem.MediaFilepath == nil {
			t.Error("expected media_filepath to be set")
		}
		if updatedMediaItem.PreventDownload {
			t.Error("expected prevent_download to be false")
		}
	})

	t.Run("calls the media_downloaded user script runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		postDownloadCalled := false
		postDownloadMediaItem := (*core.MediaItem)(nil)

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, outputTemplate string, addlOpts core.KW) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			if action == "download" {
				metadata, _ := coretest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			if action == "download_thumbnail" {
				return "", nil
			}
			return "{}", nil
		})

		ta.UserScriptMock.Run.ExpectN(2, func(event string, data any) error {
			if event == "media_pre_download" {
				return nil
			}
			if event == "media_downloaded" {
				postDownloadCalled = true
				if item, ok := data.(*core.MediaItem); ok {
					postDownloadMediaItem = item
				}
			}
			return nil
		})

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})

		_ = ta.Oban.PerformJob(ctx, core.MediaDownloadWorkerName, map[string]any{"id": mediaItem.ID})

		if !postDownloadCalled {
			t.Error("expected media_downloaded to be called")
		}
		if postDownloadMediaItem == nil || postDownloadMediaItem.ID != mediaItem.ID {
			t.Error("expected post-download script to receive the correct media item")
		}
	})
}
