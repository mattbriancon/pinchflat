package core_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestPodcastHelpers_OpmlSources(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("filters deleted sources", func(t *testing.T) {

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		coretest.SourceFixture(t, ta, core.Attrs{"marked_for_deletion_at": coretest.Now()})

		found, err := ta.PodcastHelpersOpmlSources(ta.Ctx)
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(found) != 1 {
			t.Errorf("found=%d, want 1", len(found))
		}
		if found[0].CustomName != source.CustomName {
			t.Errorf("name=%q, want %q", found[0].CustomName, source.CustomName)
		}
		if *found[0].UUID != *source.UUID {
			t.Errorf("uuid=%q, want %q", *found[0].UUID, *source.UUID)
		}
	})
}

func TestPodcastHelpers_PersistedMediaItemsFor(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("filters to existing files", func(t *testing.T) {

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		goodMedia := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID})
		coretest.MediaItemFixture(t, ta, core.Attrs{"source_id": source.ID, "media_filepath": "/tmp/existing_file.mp3"})

		persisted, err := ta.PodcastHelpersPersistedMediaItemsFor(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(persisted) != 1 {
			t.Errorf("persisted=%d, want 1", len(persisted))
		}
		if persisted[0].ID != goodMedia.ID {
			t.Errorf("id=%d, want %d", persisted[0].ID, goodMedia.ID)
		}
	})

	t.Run("applies limit", func(t *testing.T) {

		source := coretest.SourceFixture(t, ta, core.Attrs{})
		coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID})

		persisted, err := ta.PodcastHelpersPersistedMediaItemsFor(ta.Ctx, source, core.KW{core.Opt("limit", 0)})
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(persisted) != 0 {
			t.Errorf("persisted=%d with limit 0, want 0", len(persisted))
		}
	})

	t.Run("orders by newest first", func(t *testing.T) {

		source := coretest.SourceFixture(t, ta, core.Attrs{})

		oldest := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID, "uploaded_at": coretest.NowMinus(2, "day")})
		current := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID, "uploaded_at": coretest.Now()})
		older := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID, "uploaded_at": coretest.NowMinus(1, "day")})

		persisted, err := ta.PodcastHelpersPersistedMediaItemsFor(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if len(persisted) != 3 {
			t.Errorf("persisted=%d, want 3", len(persisted))
		}
		if persisted[0].ID != current.ID || persisted[1].ID != older.ID || persisted[2].ID != oldest.ID {
			t.Errorf("order: [%d,%d,%d], want [%d,%d,%d]", persisted[0].ID, persisted[1].ID, persisted[2].ID, current.ID, older.ID, oldest.ID)
		}
	})
}

func TestPodcastHelpers_SelectCoverImage(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)
	defer ta.App.DB.Close()

	t.Run("uses source poster", func(t *testing.T) {

		source := coretest.SourceWithMetadataAttachmentsFixture(t, ta, core.Attrs{})

		res, err := ta.PodcastHelpersSelectCoverImage(ta.Ctx, source, []*core.MediaItem{})
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if res != *source.Metadata.PosterFilepath {
			t.Errorf("res=%q, want %q", res, *source.Metadata.PosterFilepath)
		}
	})

	t.Run("falls back to fanart", func(t *testing.T) {

		source := coretest.SourceWithMetadataAttachmentsFixture(t, ta, core.Attrs{})
		os.Remove(*source.Metadata.PosterFilepath)

		res, err := ta.PodcastHelpersSelectCoverImage(ta.Ctx, source, []*core.MediaItem{})
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if res != *source.Metadata.FanartFilepath {
			t.Errorf("res=%q, want %q", res, *source.Metadata.FanartFilepath)
		}
	})

	t.Run("falls back to media thumbnail", func(t *testing.T) {

		source := coretest.SourceWithMetadataAttachmentsFixture(t, ta, core.Attrs{})
		mediaItem := coretest.MediaItemWithMetadataAttachmentsFixture(t, ta, core.Attrs{"source_id": source.ID})

		os.Remove(*source.Metadata.PosterFilepath)
		os.Remove(*source.Metadata.FanartFilepath)

		res, err := ta.PodcastHelpersSelectCoverImage(ta.Ctx, source, []*core.MediaItem{mediaItem})
		if err != nil {
			t.Fatalf("failed: %v", err)
		}

		if res != mediaItem.Metadata.ThumbnailFilepath {
			t.Errorf("res=%q, want %q", res, mediaItem.Metadata.ThumbnailFilepath)
		}
	})

	t.Run("errors when no artwork", func(t *testing.T) {

		source := coretest.SourceFixture(t, ta, core.Attrs{})

		_, err := ta.PodcastHelpersSelectCoverImage(ta.Ctx, source, []*core.MediaItem{})
		if err == nil {
			t.Error("expected error")
		}
	})
}
