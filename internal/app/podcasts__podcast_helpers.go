package app

import (
	"context"
	"os"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// PodcastHelpers provides helper methods for fetching podcast-related data.

// PodcastHelpersOpmlSources/0
func (a *App) PodcastHelpersOpmlSources(ctx context.Context) ([]*store.Source, error) {
	q := store.SourcesQueryNew().
		Columns("custom_name", "uuid").
		Where(sq.Eq{"s.marked_for_deletion_at": nil}).
		OrderBy("s.custom_name ASC")

	return store.All[store.Source](ctx, a.Q(ctx), q)
}

// PodcastHelpersPersistedMediaItemsFor/2
func (a *App) PodcastHelpersPersistedMediaItemsFor(ctx context.Context, source *store.Source, opts store.KW) ([]*store.MediaItem, error) {
	limit := opts.GetOr("limit", 1000).(int)

	q := store.MediaQueryNew().
		Where(store.MediaQueryForSource(source.ID)).
		Where(store.MediaQueryDownloaded()).
		Map(func(b sq.SelectBuilder) sq.SelectBuilder {
			return b.OrderBy("mi.uploaded_at DESC").Limit(uint64(limit))
		})

	mediaItems, err := store.All[store.MediaItem](ctx, a.Q(ctx), q)
	if err != nil {
		return nil, err
	}

	// Filter by file existence
	var persisted []*store.MediaItem
	for _, item := range mediaItems {
		if item.MediaFilepath != nil && *item.MediaFilepath != "" {
			if _, err := os.Stat(*item.MediaFilepath); err == nil {
				persisted = append(persisted, item)
			}
		}
	}

	return persisted, nil
}

// PodcastHelpersSelectCoverImage/2
func (a *App) PodcastHelpersSelectCoverImage(ctx context.Context, source *store.Source, mediaItems []*store.MediaItem) (string, error) {
	// Preload source with metadata
	sourceWithMetadata := source
	_, err := a.PreloadSourceMetadata(ctx, sourceWithMetadata)
	if err != nil {
		return "", err
	}

	// Get images by preference
	images := podcastHelpersGetImagesByPreference(ctx, a, sourceWithMetadata, mediaItems)

	// Find first existing file
	for _, img := range images {
		if img != "" {
			if _, err := os.Stat(img); err == nil {
				return img, nil
			}
		}
	}

	return "", store.ErrNotFound
}

// podcastHelpersGetImagesByPreference returns images in order of preference: source poster, source fanart, media item thumbnail
func podcastHelpersGetImagesByPreference(ctx context.Context, a *App, source *store.Source, mediaItems []*store.MediaItem) []string {
	var images []string

	// Add source images if metadata is loaded
	if source.Metadata != nil {
		if source.Metadata.PosterFilepath != nil {
			images = append(images, *source.Metadata.PosterFilepath)
		}
		if source.Metadata.FanartFilepath != nil {
			images = append(images, *source.Metadata.FanartFilepath)
		}
	}

	// Add media item thumbnail if available
	if len(mediaItems) > 0 {
		// Preload metadata for the first media item
		if mediaItems[0].Metadata == nil {
			a.PreloadMediaItemMetadata(ctx, mediaItems[0])
		}
		if mediaItems[0].Metadata != nil && mediaItems[0].Metadata.ThumbnailFilepath != "" {
			images = append(images, mediaItems[0].Metadata.ThumbnailFilepath)
		}
	}

	return images
}
