package coretest

import (
	"path/filepath"
	"strconv"
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
func MediaItemFixture(t testing.TB, ta *TestApp, attrs store.Attrs) *store.MediaItem {
	t.Helper()
	mediaID := randBase64(12)

	defaults := store.Attrs{
		"media_id":           mediaID,
		"title":              "Product " + randBase64(6) + " " + mediaID,
		"original_url":       "https://www.youtube.com/watch?v=" + mediaID,
		"livestream":         false,
		"short_form_content": false,
		"media_filepath":     "/video/" + randomVideoName(),
		"source_id":          SourceFixture(t, ta, store.Attrs{}).ID,
		"uploaded_at":        db.UTCDateTime{Time: time.Now().UTC()},
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	mediaItem, err := ta.CreateMediaItem(ta.Ctx, defaults)
	if err != nil {
		t.Fatalf("MediaItemFixture: %v", err)
	}
	return mediaItem
}

// MediaItemWithMetadataAttachmentsFixture creates a MediaItem with metadata attachments.
func MediaItemWithMetadataAttachmentsFixture(t testing.TB, ta *TestApp, attrs store.Attrs) *store.MediaItem {
	t.Helper()

	metadataDir := filepath.Join(ta.Config.MetadataDirectory, strconv.Itoa(randInt(1000000)+1))

	jsonGzFilepath := filepath.Join(metadataDir, "metadata.json.gz")
	thumbnailFilepath := filepath.Join(metadataDir, "thumbnail.jpg")

	if err := fileCopy(RepoPath("testdata/support/files/media_metadata.json"), jsonGzFilepath); err != nil {
		t.Fatalf("copy metadata: %v", err)
	}
	if err := fileCopy(RepoPath("testdata/support/files/thumbnail.jpg"), thumbnailFilepath); err != nil {
		t.Fatalf("copy thumbnail: %v", err)
	}

	defaults := store.Attrs{
		"metadata": map[string]string{
			"metadata_filepath":  jsonGzFilepath,
			"thumbnail_filepath": thumbnailFilepath,
		},
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	return MediaItemWithAttachmentsFixture(t, ta, defaults)
}

// MediaItemWithAttachmentsFixture creates a MediaItem with media, thumbnail, and subtitle files.
func MediaItemWithAttachmentsFixture(t testing.TB, ta *TestApp, attrs store.Attrs) *store.MediaItem {
	t.Helper()

	baseDir := filepath.Join(ta.Config.MediaDirectory, strconv.Itoa(randInt(1000000)+1))

	storedMediaFilepath := filepath.Join(baseDir, "media.mp4")
	thumbnailFilepath := filepath.Join(baseDir, "thumbnail.jpg")
	subtitleFilepath := filepath.Join(baseDir, "subtitle.en.srt")

	if err := fileCopy(RepoPath("testdata/support/files/media.mkv"), storedMediaFilepath); err != nil {
		t.Fatalf("copy media: %v", err)
	}
	if err := fileCopy(RepoPath("testdata/support/files/thumbnail.jpg"), thumbnailFilepath); err != nil {
		t.Fatalf("copy thumbnail: %v", err)
	}
	if err := fileCopy(RepoPath("testdata/support/files/subtitle.srt"), subtitleFilepath); err != nil {
		t.Fatalf("copy subtitle: %v", err)
	}

	defaults := store.Attrs{
		"media_filepath":     storedMediaFilepath,
		"thumbnail_filepath": thumbnailFilepath,
		"subtitle_filepaths": db.NestedStringArray{[]string{"en", subtitleFilepath}},
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	return MediaItemFixture(t, ta, defaults)
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
	return filepath.Join(
		RepoPath(""),
		"testdata",
		"support",
		"files",
		"media.mkv",
	)
}
