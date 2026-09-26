package core

import (
	"context"
)

// PodcastHelpers provides helper methods for fetching podcast-related data.

// PodcastHelpersOpmlSources/0
func (a *App) PodcastHelpersOpmlSources(ctx context.Context) ([]*Source, error) {
	panic("unported: Pinchflat.Podcasts.PodcastHelpers.opml_sources/0")
}

// PodcastHelpersPersistedMediaItemsFor/2
func (a *App) PodcastHelpersPersistedMediaItemsFor(ctx context.Context, source *Source, opts KW) ([]*MediaItem, error) {
	panic("unported: Pinchflat.Podcasts.PodcastHelpers.persisted_media_items_for/2")
}

// PodcastHelpersSelectCoverImage/2
func (a *App) PodcastHelpersSelectCoverImage(ctx context.Context, source *Source, mediaItems []*MediaItem) (string, error) {
	panic("unported: Pinchflat.Podcasts.PodcastHelpers.select_cover_image/2")
}
