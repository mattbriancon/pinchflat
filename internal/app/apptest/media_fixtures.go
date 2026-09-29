package apptest

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

// MediaItemFixture creates a MediaItem with sensible defaults for whatever p
// leaves unset.
func MediaItemFixture(t testing.TB, ta *TestApp, p store.MediaItemParams) *store.MediaItem {
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
		p.SourceID = store.Ptr(SourceFixture(t, ta, store.SourceParams{}).ID)
	}
	if p.UploadedAt == nil && p.Clear&store.ClearUploadedAt == 0 {
		p.UploadedAt = store.Ptr(time.Now().UTC())
	}

	mediaItem, err := ta.CreateMediaItem(ta.Ctx, p)
	if err != nil {
		t.Fatalf("MediaItemFixture: %v", err)
	}
	return mediaItem
}

// MediaItemWithMetadataAttachmentsFixture creates a MediaItem with metadata attachments.
func MediaItemWithMetadataAttachmentsFixture(t testing.TB, ta *TestApp, p store.MediaItemParams) *store.MediaItem {
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

	if p.Metadata == nil {
		p.Metadata = &store.MediaMetadataParams{
			MetadataFilepath:  &jsonGzFilepath,
			ThumbnailFilepath: &thumbnailFilepath,
		}
	}

	return MediaItemWithAttachmentsFixture(t, ta, p)
}

// MediaItemWithAttachmentsFixture creates a MediaItem with media, thumbnail, and subtitle files.
func MediaItemWithAttachmentsFixture(t testing.TB, ta *TestApp, p store.MediaItemParams) *store.MediaItem {
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

	if p.MediaFilepath == nil && p.Clear&store.ClearMediaFilepath == 0 {
		p.MediaFilepath = &storedMediaFilepath
	}
	if p.ThumbnailFilepath == nil && p.Clear&store.ClearThumbnailFilepath == 0 {
		p.ThumbnailFilepath = &thumbnailFilepath
	}
	if p.SubtitleFilepaths == nil {
		p.SubtitleFilepaths = &db.NestedStringArray{{"en", subtitleFilepath}}
	}

	return MediaItemFixture(t, ta, p)
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
