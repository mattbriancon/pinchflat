package core

import (
	"context"
	"sync"
)

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

// YoutubeApiNextAPIKey/0
func (a *App) YoutubeApiNextAPIKey(ctx context.Context) *string {
	panic("unported: Pinchflat.FastIndexing.YoutubeApi.next_api_key/0")
}

// YoutubeApiConstructAPIEndpoint/1
func YoutubeApiConstructAPIEndpoint(playlistID string, apiKey *string) string {
	panic("unported: Pinchflat.FastIndexing.YoutubeApi.construct_api_endpoint/1")
}

// YoutubeApiGetMediaIDsFromResponse/1
func YoutubeApiGetMediaIDsFromResponse(parsedJSON map[string]any) []string {
	panic("unported: Pinchflat.FastIndexing.YoutubeApi.get_media_ids_from_response/1")
}

// YoutubeApiDeterminePlaylistID/1
func YoutubeApiDeterminePlaylistID(source *Source) string {
	panic("unported: Pinchflat.FastIndexing.YoutubeApi.determine_playlist_id/1")
}

// YoutubeApiDoAPIRequest/2
func (a *App) YoutubeApiDoAPIRequest(ctx context.Context, playlistID string) (map[string]any, error) {
	panic("unported: Pinchflat.FastIndexing.YoutubeApi.do_api_request/1")
}

// YoutubeApiAPIKeys/0
func (a *App) YoutubeApiAPIKeys(ctx context.Context) []string {
	panic("unported: Pinchflat.FastIndexing.YoutubeApi.api_keys/0")
}

// YoutubeApiGetOrStartAPIKeyAgent/0
func (a *App) YoutubeApiGetOrStartAPIKeyAgent() sync.Mutex {
	panic("unported: Pinchflat.FastIndexing.YoutubeApi.get_or_start_api_key_agent/0")
}

// Module-level state for round-robin API key selection
var (
	youtubeApiKeyIndexMu sync.Mutex
	youtubeApiKeyIndex   int32 = 0
)
