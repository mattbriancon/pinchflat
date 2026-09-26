package core

import (
	"context"
)

// RssFeedBuilder builds RSS feeds for sources and their media items.

const rssFeedBuilderDatetimeFormat = "%a, %d %b %Y %H:%M:%S %z"

// RssFeedBuilderBuild/2
func (a *App) RssFeedBuilderBuild(ctx context.Context, source *Source, opts KW) (string, error) {
	panic("unported: Pinchflat.Podcasts.RssFeedBuilder.build/2")
}

// RssFeedBuilderItemImagePath/2
func RssFeedBuilderItemImagePath(urlBase string, mediaItem *MediaItem) *string {
	panic("unported: Pinchflat.Podcasts.RssFeedBuilder.item_image_path/2")
}
