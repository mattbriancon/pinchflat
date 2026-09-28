package storetest

import (
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// randomVideoName generates a random video filename.
func randomVideoName() string {
	extensions := []string{".mp4", ".mkv", ".webm", ".avi"}
	ext := extensions[randInt(len(extensions))]
	return "video_" + randBase64(8) + ext
}

// MediaItemFixture creates a MediaItem with sensible defaults, merging in attrs.
func MediaItemFixture(t testing.TB, ts *TestStore, attrs store.Attrs) *store.MediaItem {
	t.Helper()
	mediaID := randBase64(12)

	defaults := store.Attrs{
		"media_id":           mediaID,
		"title":              "Product " + randBase64(6) + " " + mediaID,
		"original_url":       "https://www.youtube.com/watch?v=" + mediaID,
		"livestream":         false,
		"short_form_content": false,
		"media_filepath":     "/video/" + randomVideoName(),
		"source_id":          SourceFixture(t, ts, store.Attrs{}).ID,
		"uploaded_at":        db.UTCDateTime{Time: time.Now().UTC()},
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	mediaItem, err := ts.CreateMediaItem(ts.Ctx, defaults)
	if err != nil {
		t.Fatalf("MediaItemFixture: %v", err)
	}
	return mediaItem
}

// MediaAttributesReturnFixture returns the JSON string matching Elixir fixture.
func MediaAttributesReturnFixture() string {
	mediaAttributes := map[string]any{
		"id":           "video1",
		"title":        "Video 1",
		"original_url": "https://example.com/video1",
		"live_status":  "not_live",
		"description":  "desc1",
		"aspect_ratio": 1.67,
		"duration":     123.45,
		"upload_date":  "20210101",
		"timestamp":    1600000000,
	}

	jsonStr, _ := db.EncodeJSON(mediaAttributes)
	return jsonStr
}

// MediaFilePathFixture returns the path to the test media file.
func MediaFilePathFixture() string {
	return RepoPath("testdata/support/files/media.mkv")
}
