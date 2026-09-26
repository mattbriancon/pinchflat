package core

import "context"

// YoutubeBehaviour defines the interface for YouTube indexing implementations.
type YoutubeBehaviour interface {
	Enabled(ctx context.Context) bool
	GetRecentMediaIDs(ctx context.Context, source *Source) ([]string, error)
}

// YoutubeApiEnabled/0
func (a *App) YoutubeApiEnabled(ctx context.Context) bool {
	panic("unported: Pinchflat.FastIndexing.YoutubeApi.enabled?/0")
}

// YoutubeApiGetRecentMediaIDs/1
func (a *App) YoutubeApiGetRecentMediaIDs(ctx context.Context, source *Source) ([]string, error) {
	panic("unported: Pinchflat.FastIndexing.YoutubeApi.get_recent_media_ids/1")
}
