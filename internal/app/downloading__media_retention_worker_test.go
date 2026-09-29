package app_test

import (
	"os"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// retentionOutcome is what a retention run should have done to an item.
type retentionOutcome struct{ culled, preventDownload bool }

// wantRetention checks an item's files and record after a retention run. item
// is the fixture (holding the original file path); a record that never had a
// media_filepath stays without one.
func wantRetention(t *testing.T, ta *apptest.TestApp, name string, item *store.MediaItem, want retentionOutcome) {
	t.Helper()
	reloaded, err := ta.GetMediaItem(ta.Ctx, item.ID)
	must(t, err)

	if item.MediaFilepath != nil {
		if exists := fileExists(*item.MediaFilepath); exists == want.culled {
			t.Errorf("%s file exists = %v, want %v", name, exists, !want.culled)
		}
	} else if want.culled {
		t.Errorf("%s has no file to cull", name)
	}
	if cleared := reloaded.MediaFilepath == nil; cleared != (want.culled || item.MediaFilepath == nil) {
		t.Errorf("%s database entry media_filepath cleared = %v, want culled = %v", name, cleared, want.culled)
	}
	if isCulled := reloaded.CulledAt != nil; isCulled != want.culled {
		t.Errorf("%s culled_at set = %v, want %v", name, isCulled, want.culled)
	} else if isCulled {
		if diff := time.Since(reloaded.CulledAt.Time); diff > time.Second {
			t.Errorf("%s culled_at should be recent, but diff is %v", name, diff)
		}
	}
	if reloaded.PreventDownload != want.preventDownload {
		t.Errorf("%s prevent_download = %v, want %v", name, reloaded.PreventDownload, want.preventDownload)
	}
}

func TestMediaRetentionWorker_Perform(t *testing.T) {
	t.Parallel()

	type items = func(*testing.T, *apptest.TestApp) (old, new *store.MediaItem)
	period := func(days *int) items {
		return func(t *testing.T, ta *apptest.TestApp) (*store.MediaItem, *store.MediaItem) {
			_, old, new := prepareRecordsForRetentionDate(t, ta, days)
			return old, new
		}
	}
	cutoff := func(days any) items {
		return func(t *testing.T, ta *apptest.TestApp) (*store.MediaItem, *store.MediaItem) {
			_, old, new := prepareRecordsForSourceCutoffDate(t, ta, days)
			return old, new
		}
	}
	update := func(t *testing.T, ta *apptest.TestApp, item *store.MediaItem, p store.MediaItemParams) *store.MediaItem {
		t.Helper()
		updated, err := ta.UpdateMediaItem(ta.Ctx, item, p)
		must(t, err)
		return updated
	}
	// tweaks change the prepared items before the worker runs
	type tweak = func(t *testing.T, ta *apptest.TestApp, old, new *store.MediaItem) (*store.MediaItem, *store.MediaItem)
	preventCulling := func(t *testing.T, ta *apptest.TestApp, old, new *store.MediaItem) (*store.MediaItem, *store.MediaItem) {
		return update(t, ta, old, store.MediaItemParams{PreventCulling: store.Ptr(true)}), new
	}
	noFilepath := func(t *testing.T, ta *apptest.TestApp, old, new *store.MediaItem) (*store.MediaItem, *store.MediaItem) {
		return update(t, ta, old, store.MediaItemParams{Clear: store.ClearMediaFilepath}), new
	}
	culled := retentionOutcome{culled: true, preventDownload: true} // by retention period
	keep := retentionOutcome{}

	for _, c := range []struct {
		name     string
		prepare  items
		tweak    tweak
		old, new retentionOutcome
	}{
		// retention period based culling
		{"culls media past the retention date, sets culled_at and prevent_download", period(store.Ptr(2)), nil, culled, keep},
		{"deletes media files that are on their retention date per the 24-h clock", period(store.Ptr(2)),
			func(t *testing.T, ta *apptest.TestApp, old, new *store.MediaItem) (*store.MediaItem, *store.MediaItem) {
				justOver := apptest.NowMinus(2, "days").Add(-1 * time.Minute)
				justUnder := apptest.NowMinus(2, "days").Add(1 * time.Minute)
				return update(t, ta, old, store.MediaItemParams{MediaDownloadedAt: store.Ptr(justOver)}),
					update(t, ta, new, store.MediaItemParams{MediaDownloadedAt: store.Ptr(justUnder)})
			}, culled, keep},
		{"doesn't cull if the source doesn't have a retention period", period(nil), nil, keep, keep},
		{"doesn't cull media items that have prevent_culling set (retention period)", period(store.Ptr(2)), preventCulling, keep, keep},
		{"doesn't cull if the media item has no media_filepath (retention period)", period(store.Ptr(2)), noFilepath, keep, keep},
		// source cutoff date based culling (culled_at but not prevent_download)
		{"culls media from before the cutoff date, sets culled_at but not prevent_download", cutoff(2), nil, retentionOutcome{culled: true}, keep},
		{"doesn't cull media from on or after the cutoff date", cutoff(2),
			func(t *testing.T, ta *apptest.TestApp, old, new *store.MediaItem) (*store.MediaItem, *store.MediaItem) {
				return update(t, ta, old, store.MediaItemParams{UploadedAt: store.Ptr(apptest.NowMinus(2, "days"))}),
					update(t, ta, new, store.MediaItemParams{UploadedAt: store.Ptr(apptest.NowMinus(1, "day"))})
			}, keep, keep},
		{"doesn't cull media if the source doesn't have a cutoff date", cutoff(nil), nil, keep, keep},
		{"doesn't cull media items that have prevent_culling set (cutoff date)", cutoff(2), preventCulling, keep, keep},
		{"doesn't cull if the media item has no media_filepath (cutoff date)", cutoff(2), noFilepath, keep, keep},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta := apptest.NewApp(t)
			ta.UserScriptMock.Run.Stub(func(event string, data any) error { return nil })
			old, new := c.prepare(t, ta)
			if c.tweak != nil {
				old, new = c.tweak(t, ta, old, new)
			}

			must(t, ta.Oban.PerformJob(ta.Ctx, app.MediaRetentionWorkerName, map[string]any{}))

			wantRetention(t, ta, "old media item", old, c.old)
			wantRetention(t, ta, "new media item", new, c.new)
		})
	}
}

// Helper functions

func prepareRecordsForRetentionDate(t testing.TB, ta *apptest.TestApp, retentionPeriodDays *int) (*store.Source, *store.MediaItem, *store.MediaItem) {
	t.Helper()

	source := apptest.SourceFixture(t, ta, store.SourceParams{RetentionPeriodDays: retentionPeriodDays})

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

	var p store.SourceParams
	if downloadCutoffDateDaysAgo != nil {
		p.DownloadCutoffDate = store.Ptr(apptest.NowMinus(downloadCutoffDateDaysAgo.(int), "days"))
	}

	source := apptest.SourceFixture(t, ta, p)

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
