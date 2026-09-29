package app_test

import (
	"os"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestMediaRetentionWorker_Perform(t *testing.T) {
	t.Parallel()

	t.Run("sets deleted media to not re-download", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForRetentionDate(t, ta, 2)

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}

		if reloadedNew.PreventDownload {
			t.Error("new media item should not have prevent_download set")
		}
		if !reloadedOld.PreventDownload {
			t.Error("old media item should have prevent_download set")
		}
	})

	t.Run("sets culled_at timestamp on deleted media", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForRetentionDate(t, ta, 2)

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}

		if reloadedNew.CulledAt != nil {
			t.Error("new media item should not have culled_at set")
		}
		if reloadedOld.CulledAt == nil {
			t.Error("old media item should have culled_at set")
		}
		if diff := time.Since(reloadedOld.CulledAt.Time); diff > time.Second {
			t.Errorf("culled_at should be recent, but diff is %v", diff)
		}
	})
}

func TestMediaRetentionWorker_Perform_WhenTestingRetentionPeriodBasedCulling(t *testing.T) {
	t.Parallel()

	t.Run("deletes media files that are past their retention date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForRetentionDate(t, ta, 2)

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should be deleted")
		}
		if reloadedNew.MediaFilepath == nil {
			t.Error("new media item database entry should still have filepath")
		}
		if reloadedOld.MediaFilepath != nil {
			t.Error("old media item database entry should have filepath cleared")
		}
	})

	t.Run("deletes media files that are on their retention date per the 24-h clock", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForRetentionDate(t, ta, 2)

		justOverTwoDaysAgo := apptest.NowMinus(2, "days").Add(-1 * time.Minute)
		justUnderTwoDaysAgo := apptest.NowMinus(2, "days").Add(1 * time.Minute)

		_, err := ta.UpdateMediaItem(ctx, oldMediaItem, store.MediaItemParams{MediaDownloadedAt: store.Ptr(justOverTwoDaysAgo)})
		if err != nil {
			t.Fatalf("update old media item: %v", err)
		}
		_, err = ta.UpdateMediaItem(ctx, newMediaItem, store.MediaItemParams{MediaDownloadedAt: store.Ptr(justUnderTwoDaysAgo)})
		if err != nil {
			t.Fatalf("update new media item: %v", err)
		}

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should be deleted")
		}
		if reloadedNew.MediaFilepath == nil {
			t.Error("new media item database entry should still have filepath")
		}
		if reloadedOld.MediaFilepath != nil {
			t.Error("old media item database entry should have filepath cleared")
		}
	})

	t.Run("sets culled_at and prevent_download", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForRetentionDate(t, ta, 2)

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}

		if reloadedNew.CulledAt != nil {
			t.Error("new media item should not have culled_at set")
		}
		if reloadedOld.CulledAt == nil {
			t.Error("old media item should have culled_at set")
		}
		if reloadedNew.PreventDownload {
			t.Error("new media item should not have prevent_download set")
		}
		if !reloadedOld.PreventDownload {
			t.Error("old media item should have prevent_download set")
		}
	})

	t.Run("doesn't cull if the source doesn't have a retention period", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForRetentionDate(t, ta, nil)

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || !fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should still exist")
		}
		if reloadedNew.MediaFilepath == nil {
			t.Error("new media item database entry should still have filepath")
		}
		if reloadedOld.MediaFilepath == nil {
			t.Error("old media item database entry should still have filepath")
		}

		if reloadedNew.CulledAt != nil {
			t.Error("new media item should not have culled_at set")
		}
		if reloadedOld.CulledAt != nil {
			t.Error("old media item should not have culled_at set")
		}
	})

	t.Run("doesn't cull media items that have prevent_culling set", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, _ := prepareRecordsForRetentionDate(t, ta, 2)

		_, err := ta.UpdateMediaItem(ctx, oldMediaItem, store.MediaItemParams{PreventCulling: store.Ptr(true)})
		if err != nil {
			t.Fatalf("update media item: %v", err)
		}

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		if oldMediaItem.MediaFilepath != nil && !fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should still exist")
		}
		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		if reloadedOld.MediaFilepath == nil {
			t.Error("old media item database entry should still have filepath")
		}
		if reloadedOld.CulledAt != nil {
			t.Error("old media item should not have culled_at set")
		}
	})

	t.Run("doesn't cull if the media item has no media_filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx
		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, _ := prepareRecordsForRetentionDate(t, ta, 2)

		_, err := ta.UpdateMediaItem(ctx, oldMediaItem, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		if err != nil {
			t.Fatalf("update media item: %v", err)
		}

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		if reloadedOld.CulledAt != nil {
			t.Error("old media item should not have culled_at set")
		}
	})
}

func TestMediaRetentionWorker_Perform_WhenTestingSourceCutoffBasedCulling(t *testing.T) {
	t.Run("culls media from before the cutoff date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForSourceCutoffDate(t, ta, 2)

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should be deleted")
		}
		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}
		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		if reloadedNew.MediaFilepath == nil {
			t.Error("new media item database entry should still have filepath")
		}
		if reloadedOld.MediaFilepath != nil {
			t.Error("old media item database entry should have filepath cleared")
		}
	})

	t.Run("doesn't cull media from on or after the cutoff date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForSourceCutoffDate(t, ta, 2)

		_, err := ta.UpdateMediaItem(ctx, oldMediaItem, store.MediaItemParams{UploadedAt: store.Ptr(apptest.NowMinus(2, "days"))})
		if err != nil {
			t.Fatalf("update old media item: %v", err)
		}
		_, err = ta.UpdateMediaItem(ctx, newMediaItem, store.MediaItemParams{UploadedAt: store.Ptr(apptest.NowMinus(1, "day"))})
		if err != nil {
			t.Fatalf("update new media item: %v", err)
		}

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}
		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || !fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should still exist")
		}
		if reloadedNew.MediaFilepath == nil {
			t.Error("new media item database entry should still have filepath")
		}
		if reloadedOld.MediaFilepath == nil {
			t.Error("old media item database entry should still have filepath")
		}

		if reloadedNew.CulledAt != nil {
			t.Error("new media item should not have culled_at set")
		}
		if reloadedOld.CulledAt != nil {
			t.Error("old media item should not have culled_at set")
		}
	})

	t.Run("sets culled_at but not prevent_download", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForSourceCutoffDate(t, ta, 2)

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}
		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}

		if reloadedNew.CulledAt != nil {
			t.Error("new media item should not have culled_at set")
		}
		if reloadedOld.CulledAt == nil {
			t.Error("old media item should have culled_at set")
		}
		if reloadedNew.PreventDownload {
			t.Error("new media item should not have prevent_download set")
		}
		if reloadedOld.PreventDownload {
			t.Error("old media item should not have prevent_download set")
		}
	})

	t.Run("doesn't cull media if the source doesn't have a cutoff date", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, newMediaItem := prepareRecordsForSourceCutoffDate(t, ta, nil)

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || !fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should still exist")
		}
		reloadedNew, err := ta.GetMediaItem(ctx, newMediaItem.ID)
		if err != nil {
			t.Fatalf("reload new media item: %v", err)
		}
		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		if reloadedNew.MediaFilepath == nil {
			t.Error("new media item database entry should still have filepath")
		}
		if reloadedOld.MediaFilepath == nil {
			t.Error("old media item database entry should still have filepath")
		}

		if reloadedNew.CulledAt != nil {
			t.Error("new media item should not have culled_at set")
		}
		if reloadedOld.CulledAt != nil {
			t.Error("old media item should not have culled_at set")
		}
	})

	t.Run("doesn't cull media items that have prevent_culling set", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, _ := prepareRecordsForSourceCutoffDate(t, ta, 2)

		_, err := ta.UpdateMediaItem(ctx, oldMediaItem, store.MediaItemParams{PreventCulling: store.Ptr(true)})
		if err != nil {
			t.Fatalf("update media item: %v", err)
		}

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		if oldMediaItem.MediaFilepath != nil && !fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should still exist")
		}
		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		if reloadedOld.MediaFilepath == nil {
			t.Error("old media item database entry should still have filepath")
		}
		if reloadedOld.CulledAt != nil {
			t.Error("old media item should not have culled_at set")
		}
	})

	t.Run("doesn't cull if the media item has no media_filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })

		_, oldMediaItem, _ := prepareRecordsForSourceCutoffDate(t, ta, 2)

		_, err := ta.UpdateMediaItem(ctx, oldMediaItem, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		if err != nil {
			t.Fatalf("update media item: %v", err)
		}

		if err := ta.Oban.PerformJob(ctx, app.MediaRetentionWorkerName, map[string]any{}); err != nil {
			t.Fatalf("PerformJob failed: %v", err)
		}

		reloadedOld, err := ta.GetMediaItem(ctx, oldMediaItem.ID)
		if err != nil {
			t.Fatalf("reload old media item: %v", err)
		}
		if reloadedOld.CulledAt != nil {
			t.Error("old media item should not have culled_at set")
		}
	})
}

// Helper functions

func prepareRecordsForRetentionDate(t testing.TB, ta *apptest.TestApp, retentionPeriodDays interface{}) (*store.Source, *store.MediaItem, *store.MediaItem) {
	t.Helper()

	attrs := store.Attrs{}
	if retentionPeriodDays != nil {
		attrs["retention_period_days"] = retentionPeriodDays
	}

	source := apptest.SourceFixture(t, ta, attrs)

	oldMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
		SourceID:          store.Ptr(source.ID),
		MediaDownloadedAt: store.Ptr(apptest.NowMinus(3, "days")),
	})

	newMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
		SourceID:          store.Ptr(source.ID),
		MediaDownloadedAt: store.Ptr(apptest.NowMinus(1, "day")),
	})

	return source, oldMediaItem, newMediaItem
}

func prepareRecordsForSourceCutoffDate(t testing.TB, ta *apptest.TestApp, downloadCutoffDateDaysAgo interface{}) (*store.Source, *store.MediaItem, *store.MediaItem) {
	t.Helper()

	attrs := store.Attrs{}
	if downloadCutoffDateDaysAgo != nil {
		cutoffDate := apptest.NowMinus(downloadCutoffDateDaysAgo.(int), "days")
		attrs["download_cutoff_date"] = &db.Date{Time: cutoffDate}
	}

	source := apptest.SourceFixture(t, ta, attrs)

	oldMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
		SourceID:   store.Ptr(source.ID),
		UploadedAt: store.Ptr(apptest.NowMinus(3, "days")),
	})

	newMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
		SourceID:   store.Ptr(source.ID),
		UploadedAt: store.Ptr(apptest.NowMinus(1, "day")),
	})

	return source, oldMediaItem, newMediaItem
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
