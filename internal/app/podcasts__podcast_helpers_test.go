package app_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestPodcastHelpers_OpmlSources(t *testing.T) {
	ta := apptest.NewApp(t)

	t.Run("returns sources not marked for deletion", func(t *testing.T) {
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		apptest.SourceFixture(t, ta, store.SourceParams{MarkedForDeletionAt: store.Ptr(apptest.Now())})

		found, err := ta.PodcastHelpersOpmlSources(ta.Ctx)
		must(t, err)

		if len(found) != 1 {
			t.Errorf("expected 1 source, got %d", len(found))
		}
		if found[0].CustomName != source.CustomName {
			t.Errorf("expected %q, got %q", source.CustomName, found[0].CustomName)
		}
		if *found[0].UUID != *source.UUID {
			t.Errorf("expected %q, got %q", *source.UUID, *found[0].UUID)
		}
	})
}

func TestPodcastHelpers_PersistedMediaItemsFor(t *testing.T) {
	ta := apptest.NewApp(t)

	t.Run("returns media items with files that exist on-disk", func(t *testing.T) {
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		goodMedia := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
		apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), MediaFilepath: store.Ptr("/tmp/existing_file.mp3")})

		persisted, err := ta.PodcastHelpersPersistedMediaItemsFor(ta.Ctx, source, 1000)
		must(t, err)

		if len(persisted) != 1 {
			t.Errorf("expected 1 media item, got %d", len(persisted))
		}
		if persisted[0].ID != goodMedia.ID {
			t.Errorf("expected ID %d, got %d", goodMedia.ID, persisted[0].ID)
		}
	})

	t.Run("lets you specify a limit", func(t *testing.T) {
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})

		persisted, err := ta.PodcastHelpersPersistedMediaItemsFor(ta.Ctx, source, 0)
		must(t, err)

		if len(persisted) != 0 {
			t.Errorf("expected 0 media items with limit 0, got %d", len(persisted))
		}
	})

	t.Run("orders by upload date where newest is first", func(t *testing.T) {
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		oldest := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), UploadedAt: store.Ptr(apptest.NowMinus(2, "day"))})
		current := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), UploadedAt: store.Ptr(apptest.Now())})
		older := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), UploadedAt: store.Ptr(apptest.NowMinus(1, "day"))})

		persisted, err := ta.PodcastHelpersPersistedMediaItemsFor(ta.Ctx, source, 1000)
		must(t, err)

		if len(persisted) != 3 {
			t.Errorf("expected 3 media items, got %d", len(persisted))
		}
		if persisted[0].ID != current.ID || persisted[1].ID != older.ID || persisted[2].ID != oldest.ID {
			t.Errorf("media items not in expected order: %d, %d, %d", persisted[0].ID, persisted[1].ID, persisted[2].ID)
		}
	})
}

func TestPodcastHelpers_SelectCoverImage(t *testing.T) {
	ta := apptest.NewApp(t)

	t.Run("returns a source's poster, if present", func(t *testing.T) {
		source := apptest.SourceWithMetadataAttachmentsFixture(t, ta, store.SourceParams{})

		res, err := ta.PodcastHelpersSelectCoverImage(ta.Ctx, source, []*store.MediaItem{})
		must(t, err)

		if res != *source.Metadata.PosterFilepath {
			t.Errorf("expected %q, got %q", *source.Metadata.PosterFilepath, res)
		}
	})

	t.Run("falls back to a source's fanart, if present", func(t *testing.T) {
		source := apptest.SourceWithMetadataAttachmentsFixture(t, ta, store.SourceParams{})

		// Remove poster file
		os.Remove(*source.Metadata.PosterFilepath)

		res, err := ta.PodcastHelpersSelectCoverImage(ta.Ctx, source, []*store.MediaItem{})
		must(t, err)

		if res != *source.Metadata.FanartFilepath {
			t.Errorf("expected %q, got %q", *source.Metadata.FanartFilepath, res)
		}
	})

	t.Run("falls back to a media item's thumbnail, if present", func(t *testing.T) {
		source := apptest.SourceWithMetadataAttachmentsFixture(t, ta, store.SourceParams{})
		mediaItem := apptest.MediaItemWithMetadataAttachmentsFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})

		// Remove source image files
		os.Remove(*source.Metadata.PosterFilepath)
		os.Remove(*source.Metadata.FanartFilepath)

		res, err := ta.PodcastHelpersSelectCoverImage(ta.Ctx, source, []*store.MediaItem{mediaItem})
		must(t, err)

		if res != mediaItem.Metadata.ThumbnailFilepath {
			t.Errorf("expected %q, got %q", mediaItem.Metadata.ThumbnailFilepath, res)
		}
	})

	t.Run("returns error if no artwork can be found", func(t *testing.T) {
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		_, err := ta.PodcastHelpersSelectCoverImage(ta.Ctx, source, []*store.MediaItem{})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
