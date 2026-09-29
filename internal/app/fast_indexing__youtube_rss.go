package app

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// YoutubeRssEnabled/0
// Determines if the YouTube RSS feed is enabled for fast indexing. Used to satisfy
// the `YoutubeBehaviour` behaviour. Always returns true.
func (a *App) YoutubeRssEnabled(ctx context.Context) bool {
	return true
}

// YoutubeRssGetRecentMediaIDs/1
// Fetches the recent media IDs from a YouTube RSS feed for a given source.
func (a *App) YoutubeRssGetRecentMediaIDs(ctx context.Context, source *store.Source) ([]string, error) {
	slog.Debug(fmt.Sprintf("Fetching recent media IDs from YouTube RSS feed for source: %s", source.CollectionID))

	url := youtubeRssURLForSource(source)
	response, err := a.HTTP.Get(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch RSS feed")
	}

	// Extract media IDs using regex
	mediaIDRegex := regexp.MustCompile(`<yt:videoId>(.*?)<\/yt:videoId>`)
	matches := mediaIDRegex.FindAllStringSubmatch(response, -1)

	mediaIDs := []string{}
	seen := make(map[string]bool)

	for _, match := range matches {
		if len(match) > 1 {
			id := strings.TrimSpace(match[1])
			if len(id) > 0 && !seen[id] {
				mediaIDs = append(mediaIDs, id)
				seen[id] = true
			}
		}
	}

	slog.Debug(fmt.Sprintf("Media ids fetched from RSS: %v", mediaIDs))

	return mediaIDs, nil
}

// youtubeRssURLForSource constructs the RSS URL for a YouTube source
func youtubeRssURLForSource(source *store.Source) string {
	switch source.CollectionType {
	case store.SourceCollectionTypeChannel:
		return fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", source.CollectionID)
	case store.SourceCollectionTypePlaylist:
		return fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?playlist_id=%s", source.CollectionID)
	default:
		return ""
	}
}
