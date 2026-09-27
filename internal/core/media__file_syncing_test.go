package core_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestFileSyncing_DeleteOutdatedFiles(t *testing.T) {
	t.Run("deletes outdated files", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})
		oldMediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})

		if err := core.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem); err != nil {
			t.Fatalf("DeleteOutdatedFiles failed: %v", err)
		}

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should be deleted")
		}
	})

	t.Run("keeps files if new matches old", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})
		oldMediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": newMediaItem.MediaFilepath})

		if err := core.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem); err != nil {
			t.Fatalf("DeleteOutdatedFiles failed: %v", err)
		}

		if newMediaItem.MediaFilepath != nil && !fileExists(*newMediaItem.MediaFilepath) {
			t.Error("new media item file should still exist")
		}
		if oldMediaItem.MediaFilepath == nil || !fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should still exist")
		}
	})

	t.Run("keeps old file if new is missing", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": nil})
		oldMediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})

		if err := core.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem); err != nil {
			t.Fatalf("DeleteOutdatedFiles failed: %v", err)
		}

		if oldMediaItem.MediaFilepath == nil || !fileExists(*oldMediaItem.MediaFilepath) {
			t.Error("old media item file should still exist")
		}
	})

	t.Run("handles subtitle files", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})
		oldMediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})

		if err := core.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem); err != nil {
			t.Fatalf("DeleteOutdatedFiles failed: %v", err)
		}

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
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})
		oldMediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"subtitle_filepaths": newMediaItem.SubtitleFilepaths})

		if err := core.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem); err != nil {
			t.Fatalf("DeleteOutdatedFiles failed: %v", err)
		}

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
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		newMediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"subtitle_filepaths": [][]string{}})
		oldMediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})

		if err := core.FileSyncingDeleteOutdatedFiles(ctx, oldMediaItem, newMediaItem); err != nil {
			t.Fatalf("DeleteOutdatedFiles failed: %v", err)
		}

		oldSubtitle := getSubtitleFilepath(oldMediaItem, "en")
		if oldSubtitle != "" && !fileExists(oldSubtitle) {
			t.Error("old subtitle file should still exist")
		}
	})
}

func TestFileSyncing_SyncFilePresenceOnDisk(t *testing.T) {
	t.Run("removes missing file attributes", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{"media_filepath": "/tmp/missing_file.mp4"})

		if mediaItem.MediaFilepath == nil {
			t.Fatal("media_filepath should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*core.MediaItem{mediaItem})
		if err != nil {
			t.Fatalf("SyncFilePresenceOnDisk failed: %v", err)
		}

		if len(updatedItems) != 1 {
			t.Fatalf("expected 1 updated item, got %d", len(updatedItems))
		}

		if updatedItems[0].MediaFilepath != nil {
			t.Error("media_filepath should be nil after sync")
		}
	})

	t.Run("keeps existing file attributes", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})

		if mediaItem.MediaFilepath == nil {
			t.Fatal("media_filepath should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*core.MediaItem{mediaItem})
		if err != nil {
			t.Fatalf("SyncFilePresenceOnDisk failed: %v", err)
		}

		if len(updatedItems) != 1 {
			t.Fatalf("expected 1 updated item")
		}

		if updatedItems[0].MediaFilepath == nil {
			t.Error("media_filepath should still be set")
		}
	})

	t.Run("touches only missing attributes", func(t *testing.T) {
		t.Parallel()
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})

		if mediaItem.MediaFilepath != nil {
			os.Remove(*mediaItem.MediaFilepath)
		}

		if mediaItem.ThumbnailFilepath == nil {
			t.Fatal("thumbnail_filepath should be set")
		}
		if mediaItem.MediaFilepath == nil {
			t.Fatal("media_filepath should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*core.MediaItem{mediaItem})
		if err != nil {
			t.Fatalf("SyncFilePresenceOnDisk failed: %v", err)
		}

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
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{
			"subtitle_filepaths": [][]string{{"en", "/tmp/missing_file.srt"}},
		})

		if len(mediaItem.SubtitleFilepaths) == 0 {
			t.Fatal("subtitle_filepaths should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*core.MediaItem{mediaItem})
		if err != nil {
			t.Fatalf("SyncFilePresenceOnDisk failed: %v", err)
		}

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
		ta := coretest.NewApp(t)
		ctx := ta.Ctx

		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})

		if len(mediaItem.SubtitleFilepaths) == 0 {
			t.Fatal("subtitle_filepaths should be set")
		}

		updatedItems, err := ta.FileSyncingSyncFilePresenceOnDisk(ctx, []*core.MediaItem{mediaItem})
		if err != nil {
			t.Fatalf("SyncFilePresenceOnDisk failed: %v", err)
		}

		if len(updatedItems) != 1 {
			t.Fatalf("expected 1 updated item")
		}

		updatedSubtitle := getSubtitleFilepath(updatedItems[0], "en")
		if updatedSubtitle == "" {
			t.Error("subtitle should still be present")
		}
	})
}

func getSubtitleFilepath(mediaItem *core.MediaItem, language string) string {
	for _, pair := range mediaItem.SubtitleFilepaths {
		if len(pair) == 2 && pair[0] == language {
			return pair[1]
		}
	}
	return ""
}
