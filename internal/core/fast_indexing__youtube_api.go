package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// YoutubeBehaviour defines the interface for YouTube indexing implementations.
type YoutubeBehaviour interface {
	Enabled(ctx context.Context) bool
	GetRecentMediaIDs(ctx context.Context, source *Source) ([]string, error)
}

var (
	youtubeApiKeyIndexMu sync.Mutex
	youtubeApiKeyIndex   int
)

// YoutubeApiEnabled/0
// Determines if the YouTube API is enabled for fast indexing by checking
// if the user has an API key set
func (a *App) YoutubeApiEnabled(ctx context.Context) bool {
	return len(youtubeApiKeys(ctx, a)) > 0
}

// YoutubeApiGetRecentMediaIDs/1
// Fetches the recent media IDs from the YouTube API for a given source.
func (a *App) YoutubeApiGetRecentMediaIDs(ctx context.Context, source *Source) ([]string, error) {
	playlistID := determinePlaylistID(source.CollectionID)
	response, err := youtubeApiRequest(ctx, a, playlistID)
	if err != nil {
		return nil, err
	}

	return youtubeApiMediaIDsFromResponse(response)
}

// determinePlaylistID replaces UC prefix with UU prefix to convert channel IDs to playlist IDs
func determinePlaylistID(collectionID string) string {
	return strings.Replace(collectionID, "UC", "UU", 1)
}

// youtubeApiRequest makes the API request and returns parsed JSON
func youtubeApiRequest(ctx context.Context, a *App, playlistID string) (map[string]interface{}, error) {
	slog.Debug(fmt.Sprintf("Fetching recent media IDs from YouTube API for playlist: %s", playlistID))

	// Get the next API key in round-robin fashion
	apiKey := youtubeApiNextKey(ctx, a)

	// Construct the URL with the API key
	url := youtubeApiEndpointWithKey(playlistID, apiKey)

	response, err := a.HTTP.Get(ctx, url, KW{Opt("accept", "application/json")}, nil)
	if err != nil {
		slog.Error(fmt.Sprintf("Failed to fetch YouTube API: %v", err))
		return nil, err
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

// youtubeApiMediaIDsFromResponse extracts media IDs from the API response
func youtubeApiMediaIDsFromResponse(parsed map[string]interface{}) ([]string, error) {
	items, ok := parsed["items"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	mediaIDs := []string{}
	seen := make(map[string]bool)

	for _, item := range items {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		contentDetails, ok := itemMap["contentDetails"].(map[string]interface{})
		if !ok {
			continue
		}

		videoID, ok := contentDetails["videoId"].(string)
		if !ok || videoID == "" {
			continue
		}

		if !seen[videoID] {
			mediaIDs = append(mediaIDs, videoID)
			seen[videoID] = true
		}
	}

	return mediaIDs, nil
}

// youtubeApiKeys retrieves the configured YouTube API keys
func youtubeApiKeys(ctx context.Context, a *App) []string {
	val, err := a.SettingsGet(ctx, "youtube_api_key")
	if err != nil || val == nil {
		return []string{}
	}

	// Handle both *string and string cases
	var keysStr string
	switch v := val.(type) {
	case *string:
		if v == nil {
			return []string{}
		}
		keysStr = *v
	case string:
		keysStr = v
	default:
		return []string{}
	}

	if keysStr == "" {
		return []string{}
	}

	keys := strings.Split(keysStr, ",")
	var result []string
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// youtubeApiNextKey returns the next API key in round-robin fashion
func youtubeApiNextKey(ctx context.Context, a *App) string {
	keys := youtubeApiKeys(ctx, a)
	if len(keys) == 0 {
		return ""
	}

	youtubeApiKeyIndexMu.Lock()
	defer youtubeApiKeyIndexMu.Unlock()

	currentIndex := youtubeApiKeyIndex % len(keys)
	youtubeApiKeyIndex = (youtubeApiKeyIndex + 1) % len(keys)

	key := keys[currentIndex]
	slog.Debug(fmt.Sprintf("Using YouTube API key: %s", key))
	return key
}

// youtubeApiEndpointWithKey constructs the YouTube API endpoint URL with the API key
func youtubeApiEndpointWithKey(playlistID, apiKey string) string {
	apiBase := "https://youtube.googleapis.com/youtube/v3/playlistItems"
	propertyType := "contentDetails"
	maxResults := 50

	return fmt.Sprintf("%s?part=%s&maxResults=%d&playlistId=%s&key=%s", apiBase, propertyType, maxResults, playlistID, apiKey)
}
