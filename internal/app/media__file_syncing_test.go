package app_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestFileSyncing_DeleteOutdatedFiles(t *testing.T) {
	t.Run("deletes outdated files", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})
		oldMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		must(t, app.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem))

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should be deleted")
		}
	})

	t.Run("keeps files if new matches old", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})
		oldMediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{MediaFilepath: newMediaItem.MediaFilepath})

		must(t, app.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem))

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || !fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should still exist")
		}
	})

	t.Run("keeps old file if new is missing", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		oldMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		must(t, app.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem))

		if oldMediaItem.MediaFilepath == nil || !fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should still exist")
		}
	})

	t.Run("handles subtitle files", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})
		oldMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		must(t, app.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem))

		newSubtitle := getSubtitleFilepath(newMediaItem, "en")
		oldSubtitle := getSubtitleFilepath(oldMediaItem, "en")

		if newSubtitle != "" && !fileExists(newSubtitle) {
			t.Error("new subtitle file should still exist")
		}
		if oldSubtitle != "" && fileExists(oldSubtitle) {
			t.Error("old subtitle file should be deleted")
		}
	})

	t.Run("keeps subtitles if new matches old", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})
		oldMediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SubtitleFilepaths: &newMediaItem.SubtitleFilepaths})

		must(t, app.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem))

		newSubtitle := getSubtitleFilepath(newMediaItem, "en")
		oldSubtitle := getSubtitleFilepath(oldMediaItem, "en")

		if newSubtitle != "" && !fileExists(newSubtitle) {
			t.Error("new subtitle file should still exist")
		}
		if oldSubtitle != "" && !fileExists(oldSubtitle) {
			t.Error("old subtitle file should still exist")
		}
	})

	t.Run("keeps old subtitles if new is missing", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SubtitleFilepaths: &db.NestedStringArray{}})
		oldMediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		must(t, app.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem))

		oldSubtitle := getSubtitleFilepath(oldMediaItem, "en")
		if oldSubtitle != "" && !fileExists(oldSubtitle) {
			t.Error("old subtitle file should still exist")
		}
	})
}

func TestFileSyncing_SyncFilePresenceOnDisk(t *testing.T) {
	t.Run("removes missing file attributes", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{MediaFilepath: store.Ptr("/tmp/missing_file.mp4")})

		if mediaItem.MediaFilepath == nil {
			t.Fatal("media_filepath should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*store.MediaItem{mediaItem})
		must(t, err)

		if len(updatedItems) != 1 {
			t.Fatalf("expected 1 updated item, got %d", len(updatedItems))
		}

		if updatedItems[0].MediaFilepath != nil {
			t.Error("media_filepath should be nil after sync")
		}
	})

	t.Run("keeps existing file attributes", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		if mediaItem.MediaFilepath == nil {
			t.Fatal("media_filepath should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*store.MediaItem{mediaItem})
		must(t, err)

		if len(updatedItems) != 1 {
			t.Fatalf("expected 1 updated item")
		}

		if updatedItems[0].MediaFilepath == nil {
			t.Error("media_filepath should still be set")
		}
	})

	t.Run("touches only missing attributes", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		if mediaItem.MediaFilepath != nil {
			os.Remove(*mediaItem.MediaFilepath)
		}

		if mediaItem.ThumbnailFilepath == nil {
			t.Fatal("thumbnail_filepath should be set")
		}
		if mediaItem.MediaFilepath == nil {
			t.Fatal("media_filepath should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*store.MediaItem{mediaItem})
		must(t, err)

		if len(updatedItems) != 1 {
			t.Fatalf("expected 1 updated item")
		}

		if updatedItems[0].ThumbnailFilepath == nil {
			t.Error("thumbnail_filepath should still be set")
		}
		if updatedItems[0].MediaFilepath != nil {
			t.Error("media_filepath should be nil")
		}
	})

	t.Run("handles missing subtitle files", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{
			SubtitleFilepaths: &db.NestedStringArray{{"en", "/tmp/missing_file.srt"}},
		})

		if len(mediaItem.SubtitleFilepaths) == 0 {
			t.Fatal("subtitle_filepaths should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*store.MediaItem{mediaItem})
		must(t, err)

		if len(updatedItems) != 1 {
			t.Fatalf("expected 1 updated item")
		}

		updatedSubtitle := getSubtitleFilepath(updatedItems[0], "en")
		if updatedSubtitle != "" {
			t.Error("subtitle should be removed")
		}
	})

	t.Run("keeps existing subtitle files", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{})

		if len(mediaItem.SubtitleFilepaths) == 0 {
			t.Fatal("subtitle_filepaths should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*store.MediaItem{mediaItem})
		must(t, err)

		if len(updatedItems) != 1 {
			t.Fatalf("expected 1 updated item")
		}

		updatedSubtitle := getSubtitleFilepath(updatedItems[0], "en")
		if updatedSubtitle == "" {
			t.Error("subtitle should still be present")
		}
	})
}

func getSubtitleFilepath(mediaItem *store.MediaItem, language string) string {
	for _, pair := range mediaItem.SubtitleFilepaths {
		if len(pair) == 2 && pair[0] == language {
			return pair[1]
		}
	}
	return ""
}
