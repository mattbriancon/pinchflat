package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// Helper to load and parse the metadata fixture
func loadMetadataFixture(t *testing.T) map[string]any {
	t.Helper()
	root := dbtest.RepoRoot()
	fixtureFile := filepath.Join(root, "testdata", "support", "files", "media_metadata.json")

	data, err := os.ReadFile(fixtureFile)
	must(t, err)

	var metadata map[string]any
	must(t, json.Unmarshal(data, &metadata))

	return metadata
}

// Helper to get a string value from metadata
func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func TestMetadataParser_ParseForMediaItem_WhenTestingMediaMetadata(t *testing.T) {
	metadata := loadMetadataFixture(t)

	t.Run("it extracts the media filepath", func(t *testing.T) {
		t.Parallel()
		result := parseMedia(t, metadata)

		mediaFilepath := result.MediaFilepath
		if mediaFilepath == nil {
			t.Fatal("expected media_filepath to not be nil")
		}

		if !strings.Contains(*mediaFilepath, "Pinchflat Example Video-ABC123") {
			t.Errorf("expected media_filepath to contain 'Pinchflat Example Video-ABC123', got %q", *mediaFilepath)
		}
		if !strings.HasSuffix(*mediaFilepath, ".mkv") {
			t.Errorf("expected media_filepath to end with '.mkv', got %q", *mediaFilepath)
		}
	})

	for _, c := range []struct {
		field, key string
		get        func(store.MediaItemParams) *string
	}{
		{"title", "title", func(p store.MediaItemParams) *string { return p.Title }},
		{"description", "description", func(p store.MediaItemParams) *string { return p.Description }},
		{"original_url", "original_url", func(p store.MediaItemParams) *string { return p.OriginalURL }},
		{"media_id", "id", func(p store.MediaItemParams) *string { return p.MediaID }},
	} {
		t.Run("it extracts the "+c.field, func(t *testing.T) {
			t.Parallel()
			if got, want := *c.get(parseMedia(t, metadata)), getString(metadata, c.key); got != want {
				t.Errorf("got %s %q, want %q", c.field, got, want)
			}
		})
	}

	t.Run("it extracts the livestream flag", func(t *testing.T) {
		t.Parallel()
		result := parseMedia(t, metadata)

		livestream := *result.Livestream

		liveStatus := getString(metadata, "live_status")
		if liveStatus != "not_live" {
			t.Errorf("test setup error: expected live_status to be 'not_live', got %q", liveStatus)
		}
		if livestream != false {
			t.Errorf("got livestream %v, want false", livestream)
		}
	})

	t.Run("the livestream flag defaults to false", func(t *testing.T) {
		t.Parallel()
		metadataCopy := cloneMap(metadata)
		metadataCopy["live_status"] = nil

		result := parseMedia(t, metadataCopy)

		livestream := *result.Livestream

		if livestream != false {
			t.Errorf("got livestream %v, want false", livestream)
		}
	})

	t.Run("it extracts the duration in seconds", func(t *testing.T) {
		t.Parallel()
		result := parseMedia(t, metadata)

		durationSecondsPtr := result.DurationSeconds
		if durationSecondsPtr == nil {
			t.Fatal("expected duration_seconds to not be nil")
		}

		duration, ok := metadata["duration"].(float64)
		if !ok {
			t.Fatalf("expected metadata duration to be float64, got %T", metadata["duration"])
		}

		expectedDuration := int(duration)
		if *durationSecondsPtr != expectedDuration {
			t.Errorf("got duration_seconds %d, want %d", *durationSecondsPtr, expectedDuration)
		}
	})
}

func TestMetadataParser_ParseForMediaItem_WhenTestingSubtitleMetadata(t *testing.T) {
	metadata := loadMetadataFixture(t)

	t.Run("extracts the subtitle filepaths", func(t *testing.T) {
		t.Parallel()
		result := parseMedia(t, metadata)

		subtitleFilepaths := *result.SubtitleFilepaths

		if len(subtitleFilepaths) < 2 {
			t.Fatalf("expected at least 2 subtitle filepaths, got %d", len(subtitleFilepaths))
		}

		if subtitleFilepaths[0][0] != "de" {
			t.Errorf("expected first language to be 'de', got %q", subtitleFilepaths[0][0])
		}
		if subtitleFilepaths[1][0] != "en" {
			t.Errorf("expected second language to be 'en', got %q", subtitleFilepaths[1][0])
		}

		germanFilepath := subtitleFilepaths[0][1]
		englishFilepath := subtitleFilepaths[1][1]

		if !strings.HasSuffix(englishFilepath, ".en.srt") {
			t.Errorf("expected english filepath to end with '.en.srt', got %q", englishFilepath)
		}
		if !strings.HasSuffix(germanFilepath, ".de.srt") {
			t.Errorf("expected german filepath to end with '.de.srt', got %q", germanFilepath)
		}
	})

	t.Run("sorts the subtitle filepaths by language", func(t *testing.T) {
		t.Parallel()
		metadataCopy := cloneMap(metadata)

		metadataCopy["requested_subtitles"] = map[string]any{
			"en": map[string]any{"filepath": "en.srt"},
			"za": map[string]any{"filepath": "za.srt"},
			"de": map[string]any{"filepath": "de.srt"},
			"al": map[string]any{"filepath": "al.srt"},
		}

		result := parseMedia(t, metadataCopy)

		subtitleFilepaths := *result.SubtitleFilepaths

		if len(subtitleFilepaths) != 4 {
			t.Fatalf("expected 4 subtitle filepaths, got %d", len(subtitleFilepaths))
		}

		expectedOrder := []string{"al", "de", "en", "za"}
		for i, lang := range expectedOrder {
			if subtitleFilepaths[i][0] != lang {
				t.Errorf("position %d: expected language %q, got %q", i, lang, subtitleFilepaths[i][0])
			}
		}
	})

	t.Run("doesn't freak out if the media has no subtitles", func(t *testing.T) {
		t.Parallel()
		metadataCopy := cloneMap(metadata)
		metadataCopy["requested_subtitles"] = map[string]any{}

		result := parseMedia(t, metadataCopy)

		subtitleFilepaths := *result.SubtitleFilepaths

		if len(subtitleFilepaths) != 0 {
			t.Errorf("expected empty subtitle_filepaths, got %d items", len(subtitleFilepaths))
		}
	})

	t.Run("doesn't freak out if the requested_subtitles key is missing", func(t *testing.T) {
		t.Parallel()
		metadataCopy := cloneMap(metadata)
		delete(metadataCopy, "requested_subtitles")

		result := parseMedia(t, metadataCopy)

		subtitleFilepaths := *result.SubtitleFilepaths

		if len(subtitleFilepaths) != 0 {
			t.Errorf("expected empty subtitle_filepaths, got %d items", len(subtitleFilepaths))
		}
	})
}

func TestMetadataParser_ParseForMediaItem_WhenTestingThumbnailMetadata(t *testing.T) {
	metadata := loadMetadataFixture(t)
	root := dbtest.RepoRoot()

	// Setup: find the thumbnail filepath and copy the fixture file there
	var originalThumbnailFilepath string
	if thumbnails, ok := metadata["thumbnails"].([]any); ok {
		for i := len(thumbnails) - 1; i >= 0; i-- {
			if thumbMap, ok := thumbnails[i].(map[string]any); ok {
				if fp, ok := thumbMap["filepath"].(string); ok {
					originalThumbnailFilepath = fp
					break
				}
			}
		}
	}

	if originalThumbnailFilepath != "" {
		modified := applyThumbnailWorkaroundForTest(originalThumbnailFilepath)

		// Create parent directories
		os.MkdirAll(filepath.Dir(modified), 0755)

		// Copy the fixture file
		fixtureFile := filepath.Join(root, "testdata", "support", "files", "thumbnail.jpg")
		fixtureData, err := os.ReadFile(fixtureFile)
		if err != nil {
			t.Skipf("skipping thumbnail tests: could not read fixture file: %v", err)
		}
		err = os.WriteFile(modified, fixtureData, 0644)
		if err != nil {
			t.Skipf("skipping thumbnail tests: could not copy fixture file: %v", err)
		}
		defer os.Remove(modified)
	}

	t.Run("extracts the thumbnail filepath", func(t *testing.T) {
		result := parseMedia(t, metadata)

		thumbnailPath := result.ThumbnailFilepath

		if thumbnailPath == nil {
			t.Fatal("expected thumbnail_filepath to not be nil")
		}

		if !strings.HasSuffix(*thumbnailPath, ".webp") {
			t.Errorf("expected thumbnail_filepath to end with '.webp', got %q", *thumbnailPath)
		}
	})

	t.Run("automatically appends `-thumb` to the thumbnail filename", func(t *testing.T) {
		result := parseMedia(t, metadata)

		thumbnailPath := result.ThumbnailFilepath

		if thumbnailPath == nil {
			t.Fatal("expected thumbnail_filepath to not be nil")
		}

		if !strings.Contains(*thumbnailPath, "-thumb.webp") {
			t.Errorf("expected thumbnail_filepath to contain '-thumb.webp', got %q", *thumbnailPath)
		}
	})

	t.Run("doesn't include thumbnail if the file doesn't exist on-disk", func(t *testing.T) {
		t.Parallel()
		metadataCopy := cloneMap(metadata)

		// Remove the file
		if originalThumbnailFilepath != "" {
			modified := applyThumbnailWorkaroundForTest(originalThumbnailFilepath)
			os.Remove(modified)
		}

		result := parseMedia(t, metadataCopy)

		thumbnailPath := result.ThumbnailFilepath
		if thumbnailPath != nil {
			t.Errorf("expected thumbnail_filepath to be nil, got %v", thumbnailPath)
		}
	})

	t.Run("doesn't freak out if the media has no thumbnails", func(t *testing.T) {
		t.Parallel()
		metadataCopy := cloneMap(metadata)
		metadataCopy["thumbnails"] = map[string]any{}

		result := parseMedia(t, metadataCopy)

		thumbnailPath := result.ThumbnailFilepath
		if thumbnailPath != nil {
			t.Errorf("expected thumbnail_filepath to be nil, got %v", thumbnailPath)
		}
	})

	t.Run("doesn't freak out if the thumbnails key is missing", func(t *testing.T) {
		t.Parallel()
		metadataCopy := cloneMap(metadata)
		delete(metadataCopy, "thumbnails")

		result := parseMedia(t, metadataCopy)

		thumbnailPath := result.ThumbnailFilepath
		if thumbnailPath != nil {
			t.Errorf("expected thumbnail_filepath to be nil, got %v", thumbnailPath)
		}
	})
}

func TestMetadataParser_ParseForMediaItem_WhenTestingInfojsonMetadata(t *testing.T) {
	metadata := loadMetadataFixture(t)
	root := dbtest.RepoRoot()

	// Setup: create the infojson file
	infojsonFilename, ok := metadata["infojson_filename"].(string)
	if !ok || infojsonFilename == "" {
		t.Skipf("skipping infojson tests: infojson_filename not found in metadata")
	}

	os.MkdirAll(filepath.Dir(infojsonFilename), 0755)

	fixtureFile := filepath.Join(root, "testdata", "support", "files", "example.info.json")
	fixtureData, err := os.ReadFile(fixtureFile)
	if err != nil {
		t.Skipf("skipping infojson tests: could not read fixture file: %v", err)
	}
	err = os.WriteFile(infojsonFilename, fixtureData, 0644)
	if err != nil {
		t.Skipf("skipping infojson tests: could not copy fixture file: %v", err)
	}
	defer os.Remove(infojsonFilename)

	t.Run("extracts the metadata filepath", func(t *testing.T) {
		result := parseMedia(t, metadata)

		metadataPath := result.MetadataFilepath

		if metadataPath == nil {
			t.Fatal("expected metadata_filepath to not be nil")
		}

		if !strings.HasSuffix(*metadataPath, ".info.json") {
			t.Errorf("expected metadata_filepath to end with '.info.json', got %q", *metadataPath)
		}
	})

	t.Run("doesn't include metadata if the file doesn't exist on-disk", func(t *testing.T) {
		t.Parallel()
		metadataCopy := cloneMap(metadata)

		os.Remove(infojsonFilename)

		result := parseMedia(t, metadataCopy)

		metadataPath := result.MetadataFilepath
		if metadataPath != nil {
			t.Errorf("expected metadata_filepath to be nil, got %v", metadataPath)
		}
	})

	t.Run("doesn't freak out if the media has no infojson", func(t *testing.T) {
		t.Parallel()
		metadataCopy := cloneMap(metadata)
		metadataCopy["infojson_filename"] = nil

		result := parseMedia(t, metadataCopy)

		metadataPath := result.MetadataFilepath
		if metadataPath != nil {
			t.Errorf("expected metadata_filepath to be nil, got %v", metadataPath)
		}
	})
}

// Helper functions

// applyThumbnailWorkaroundForTest replicates the thumbnail workaround logic
func applyThumbnailWorkaroundForTest(filepath string) string {
	lastDotIdx := -1
	secondLastDotIdx := -1

	for i := len(filepath) - 1; i >= 0; i-- {
		if filepath[i] == '.' {
			if lastDotIdx == -1 {
				lastDotIdx = i
			} else if secondLastDotIdx == -1 {
				secondLastDotIdx = i
				break
			}
		}
	}

	if lastDotIdx == -1 {
		return filepath + "-thumb"
	}

	if secondLastDotIdx == -1 {
		return filepath[:lastDotIdx] + "-thumb" + filepath[lastDotIdx:]
	}

	return filepath[:secondLastDotIdx] + "-thumb" + filepath[secondLastDotIdx:]
}

func cloneMap(m map[string]any) map[string]any {
	c := make(map[string]any, len(m))
	for k, v := range m {
		c[k] = v
	}
	return c
}

func parseMedia(t *testing.T, metadata map[string]any) store.MediaItemParams {
	t.Helper()
	result, err := app.MetadataParserParseForMediaItem(metadata)
	must(t, err)
	return result
}
