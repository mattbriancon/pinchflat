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

// MediaItemFixture creates a MediaItem with sensible defaults for whatever p
// leaves unset.
func MediaItemFixture(t testing.TB, ts *TestStore, p store.MediaItemParams) *store.MediaItem {
	t.Helper()
	mediaID := randBase64(12)

	if p.MediaID == nil {
		p.MediaID = &mediaID
	}
	if p.Title == nil && p.Clear&store.ClearTitle == 0 {
		p.Title = store.Ptr("Product " + randBase64(6) + " " + *p.MediaID)
	}
	if p.OriginalURL == nil {
		p.OriginalURL = store.Ptr("https://www.youtube.com/watch?v=" + *p.MediaID)
	}
	if p.Livestream == nil {
		p.Livestream = store.Ptr(false)
	}
	if p.ShortFormContent == nil && p.Clear&store.ClearShortFormContent == 0 {
		p.ShortFormContent = store.Ptr(false)
	}
	if p.MediaFilepath == nil && p.Clear&store.ClearMediaFilepath == 0 {
		p.MediaFilepath = store.Ptr("/video/" + randomVideoName())
	}
	if p.SourceID == nil {
		p.SourceID = store.Ptr(SourceFixture(t, ts, store.SourceParams{}).ID)
	}
	if p.UploadedAt == nil && p.Clear&store.ClearUploadedAt == 0 {
		p.UploadedAt = store.Ptr(time.Now().UTC())
	}

	mediaItem, err := ts.CreateMediaItem(ts.Ctx, p)
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
