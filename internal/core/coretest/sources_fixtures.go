package coretest

import (
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db"
)

const (
	base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
)

// randBase64 generates a pseudo-random base64 string of the given length.
func randBase64(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = base64Chars[rand.Intn(len(base64Chars))]
	}
	return string(b)
}

// randInt generates a pseudo-random integer up to max (exclusive).
func randInt(max int) int {
	return rand.Intn(max)
}

// fileCopy copies a file from src to dst, creating parent directories if needed.
func fileCopy(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// SourceFixture creates a Source with sensible defaults, merging in attrs.
func SourceFixture(t testing.TB, ta *TestApp, attrs core.Attrs) *core.Source {
	t.Helper()
	defaults := core.Attrs{
		"enabled":                 true,
		"collection_name":         "Source #" + strconv.Itoa(randInt(1000000)+1),
		"collection_id":           randBase64(12),
		"collection_type":         "channel",
		"custom_name":             "Cool and good internal name!",
		"description":             "This is a description",
		"original_url":            "https://www.youtube.com/@" + randBase64(12),
		"media_profile_id":        MediaProfileFixture(t, ta, core.Attrs{}).ID,
		"index_frequency_minutes": 60,
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	// Like Elixir: Source.changeset(:pre_insert) |> Repo.insert(), not
	// Sources.create_source (which would kick off indexing jobs).
	source, err := core.Insert[core.Source](ta.Ctx, ta.Q(ta.Ctx), core.SourceChangeset(core.NewSource(), defaults, "pre_insert"))
	if err != nil {
		t.Fatalf("SourceFixture: %v", err)
	}
	return source
}

// SourceWithMetadataFixture creates a Source with metadata.
func SourceWithMetadataFixture(t testing.TB, ta *TestApp, attrs core.Attrs) *core.Source {
	t.Helper()
	defaults := core.Attrs{
		"metadata": map[string]string{
			"metadata_filepath": filepath.Join(ta.Config.MetadataDirectory, "metadata.json.gz"),
		},
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	return SourceFixture(t, ta, defaults)
}

// SourceWithMetadataAttachmentsFixture creates a Source with metadata attachments.
func SourceWithMetadataAttachmentsFixture(t testing.TB, ta *TestApp, attrs core.Attrs) *core.Source {
	t.Helper()

	metadataDir := filepath.Join(ta.Config.MetadataDirectory, strconv.Itoa(randInt(1000000)+1))

	jsonGzFilepath := filepath.Join(metadataDir, "metadata.json.gz")
	posterFilepath := filepath.Join(metadataDir, "poster.jpg")
	fanartFilepath := filepath.Join(metadataDir, "fanart.jpg")

	// Copy test files
	if err := fileCopy(RepoPath("test/support/files/media_metadata.json"), jsonGzFilepath); err != nil {
		t.Fatalf("copy metadata: %v", err)
	}
	if err := fileCopy(RepoPath("test/support/files/thumbnail.jpg"), posterFilepath); err != nil {
		t.Fatalf("copy poster: %v", err)
	}
	if err := fileCopy(RepoPath("test/support/files/thumbnail.jpg"), fanartFilepath); err != nil {
		t.Fatalf("copy fanart: %v", err)
	}

	defaults := core.Attrs{
		"metadata": map[string]string{
			"metadata_filepath": jsonGzFilepath,
			"poster_filepath":   posterFilepath,
			"fanart_filepath":   fanartFilepath,
		},
	}

	// Merge attrs into defaults
	for k, v := range attrs {
		defaults[k] = v
	}

	return SourceFixture(t, ta, defaults)
}

// SourceAttributesReturnFixture returns the JSON string matching Elixir fixture.
func SourceAttributesReturnFixture() string {
	sourceAttributes := []map[string]any{
		{
			"id":           "video1",
			"title":        "Video 1",
			"original_url": "https://example.com/video1",
			"live_status":  "not_live",
			"description":  "desc1",
			"aspect_ratio": 1.67,
			"duration":     12.34,
			"upload_date":  "20210101",
		},
		{
			"id":           "video2",
			"title":        "Video 2",
			"original_url": "https://example.com/video2",
			"live_status":  "is_live",
			"description":  "desc2",
			"aspect_ratio": 1.67,
			"duration":     345.67,
			"upload_date":  "20220202",
		},
		{
			"id":           "video3",
			"title":        "Video 3",
			"original_url": "https://example.com/video3",
			"live_status":  "not_live",
			"description":  "desc3",
			"aspect_ratio": 1.0,
			"duration":     678.90,
			"upload_date":  "20230303",
		},
	}

	var result string
	for i, attr := range sourceAttributes {
		if i > 0 {
			result += "\n"
		}
		jsonBytes, _ := db.EncodeJSON(attr)
		result += jsonBytes
	}
	return result
}

// SourceDetailsReturnFixture returns the JSON string for source details.
func SourceDetailsReturnFixture(attrs core.Attrs) string {
	channelID := randBase64(12)

	defaults := map[string]any{
		"channel_id":     channelID,
		"channel":        "Channel Name",
		"playlist_id":    channelID,
		"playlist_title": "Channel Name",
		"filename":       filepath.Join(os.Getenv("PINCHFLAT_MEDIA_DIR"), "foo", "bar.mp4"),
	}

	// Merge attrs
	for k, v := range attrs {
		defaults[k] = v
	}

	jsonStr, _ := db.EncodeJSON(defaults)
	return jsonStr
}
