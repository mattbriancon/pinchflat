package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
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
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

		mediaFilepath := result.MediaFilepath
		if mediaFilepath == nil {
			t.Fatal("expected media_filepath to not be nil")
		}

		if !contains(*mediaFilepath, "Pinchflat Example Video-ABC123") {
			t.Errorf("expected media_filepath to contain 'Pinchflat Example Video-ABC123', got %q", *mediaFilepath)
		}
		if !endsWith(*mediaFilepath, ".mkv") {
			t.Errorf("expected media_filepath to end with '.mkv', got %q", *mediaFilepath)
		}
	})

	t.Run("it extracts the title", func(t *testing.T) {
		t.Parallel()
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

		title := *result.Title

		expectedTitle := getString(metadata, "title")
		if title != expectedTitle {
			t.Errorf("got title %q, want %q", title, expectedTitle)
		}
	})

	t.Run("it extracts the description", func(t *testing.T) {
		t.Parallel()
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

		description := *result.Description

		expectedDescription := getString(metadata, "description")
		if description != expectedDescription {
			t.Errorf("got description %q, want %q", description, expectedDescription)
		}
	})

	t.Run("it extracts the original_url", func(t *testing.T) {
		t.Parallel()
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

		originalURL := *result.OriginalURL

		expectedURL := getString(metadata, "original_url")
		if originalURL != expectedURL {
			t.Errorf("got original_url %q, want %q", originalURL, expectedURL)
		}
	})

	t.Run("it extracts the media_id", func(t *testing.T) {
		t.Parallel()
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

		mediaID := *result.MediaID

		expectedID := getString(metadata, "id")
		if mediaID != expectedID {
			t.Errorf("got media_id %q, want %q", mediaID, expectedID)
		}
	})

	t.Run("it extracts the livestream flag", func(t *testing.T) {
		t.Parallel()
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

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
		metadataCopy := make(map[string]any)
		for k, v := range metadata {
			metadataCopy[k] = v
		}
		metadataCopy["live_status"] = nil

		result, err := app.MetadataParserParseForMediaItem(metadataCopy)
		must(t, err)

		livestream := *result.Livestream

		if livestream != false {
			t.Errorf("got livestream %v, want false", livestream)
		}
	})

	t.Run("it extracts the duration in seconds", func(t *testing.T) {
		t.Parallel()
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

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
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

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

		if !endsWith(englishFilepath, ".en.srt") {
			t.Errorf("expected english filepath to end with '.en.srt', got %q", englishFilepath)
		}
		if !endsWith(germanFilepath, ".de.srt") {
			t.Errorf("expected german filepath to end with '.de.srt', got %q", germanFilepath)
		}
	})

	t.Run("sorts the subtitle filepaths by language", func(t *testing.T) {
		t.Parallel()
		metadataCopy := make(map[string]any)
		for k, v := range metadata {
			metadataCopy[k] = v
		}

		metadataCopy["requested_subtitles"] = map[string]any{
			"en": map[string]any{"filepath": "en.srt"},
			"za": map[string]any{"filepath": "za.srt"},
			"de": map[string]any{"filepath": "de.srt"},
			"al": map[string]any{"filepath": "al.srt"},
		}

		result, err := app.MetadataParserParseForMediaItem(metadataCopy)
		must(t, err)

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
		metadataCopy := make(map[string]any)
		for k, v := range metadata {
			metadataCopy[k] = v
		}
		metadataCopy["requested_subtitles"] = map[string]any{}

		result, err := app.MetadataParserParseForMediaItem(metadataCopy)
		must(t, err)

		subtitleFilepaths := *result.SubtitleFilepaths

		if len(subtitleFilepaths) != 0 {
			t.Errorf("expected empty subtitle_filepaths, got %d items", len(subtitleFilepaths))
		}
	})

	t.Run("doesn't freak out if the requested_subtitles key is missing", func(t *testing.T) {
		t.Parallel()
		metadataCopy := make(map[string]any)
		for k, v := range metadata {
			metadataCopy[k] = v
		}
		delete(metadataCopy, "requested_subtitles")

		result, err := app.MetadataParserParseForMediaItem(metadataCopy)
		must(t, err)

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
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

		thumbnailPath := result.ThumbnailFilepath

		if thumbnailPath == nil {
			t.Fatal("expected thumbnail_filepath to not be nil")
		}

		if !endsWith(*thumbnailPath, ".webp") {
			t.Errorf("expected thumbnail_filepath to end with '.webp', got %q", *thumbnailPath)
		}
	})

	t.Run("automatically appends `-thumb` to the thumbnail filename", func(t *testing.T) {
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

		thumbnailPath := result.ThumbnailFilepath

		if thumbnailPath == nil {
			t.Fatal("expected thumbnail_filepath to not be nil")
		}

		if !contains(*thumbnailPath, "-thumb.webp") {
			t.Errorf("expected thumbnail_filepath to contain '-thumb.webp', got %q", *thumbnailPath)
		}
	})

	t.Run("doesn't include thumbnail if the file doesn't exist on-disk", func(t *testing.T) {
		t.Parallel()
		metadataCopy := make(map[string]any)
		for k, v := range metadata {
			metadataCopy[k] = v
		}

		// Remove the file
		if originalThumbnailFilepath != "" {
			modified := applyThumbnailWorkaroundForTest(originalThumbnailFilepath)
			os.Remove(modified)
		}

		result, err := app.MetadataParserParseForMediaItem(metadataCopy)
		must(t, err)

		thumbnailPath := result.ThumbnailFilepath
		if thumbnailPath != nil {
			t.Errorf("expected thumbnail_filepath to be nil, got %v", thumbnailPath)
		}
	})

	t.Run("doesn't freak out if the media has no thumbnails", func(t *testing.T) {
		t.Parallel()
		metadataCopy := make(map[string]any)
		for k, v := range metadata {
			metadataCopy[k] = v
		}
		metadataCopy["thumbnails"] = map[string]any{}

		result, err := app.MetadataParserParseForMediaItem(metadataCopy)
		must(t, err)

		thumbnailPath := result.ThumbnailFilepath
		if thumbnailPath != nil {
			t.Errorf("expected thumbnail_filepath to be nil, got %v", thumbnailPath)
		}
	})

	t.Run("doesn't freak out if the thumbnails key is missing", func(t *testing.T) {
		t.Parallel()
		metadataCopy := make(map[string]any)
		for k, v := range metadata {
			metadataCopy[k] = v
		}
		delete(metadataCopy, "thumbnails")

		result, err := app.MetadataParserParseForMediaItem(metadataCopy)
		must(t, err)

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
		result, err := app.MetadataParserParseForMediaItem(metadata)
		must(t, err)

		metadataPath := result.MetadataFilepath

		if metadataPath == nil {
			t.Fatal("expected metadata_filepath to not be nil")
		}

		if !endsWith(*metadataPath, ".info.json") {
			t.Errorf("expected metadata_filepath to end with '.info.json', got %q", *metadataPath)
		}
	})

	t.Run("doesn't include metadata if the file doesn't exist on-disk", func(t *testing.T) {
		t.Parallel()
		metadataCopy := make(map[string]any)
		for k, v := range metadata {
			metadataCopy[k] = v
		}

		os.Remove(infojsonFilename)

		result, err := app.MetadataParserParseForMediaItem(metadataCopy)
		must(t, err)

		metadataPath := result.MetadataFilepath
		if metadataPath != nil {
			t.Errorf("expected metadata_filepath to be nil, got %v", metadataPath)
		}
	})

	t.Run("doesn't freak out if the media has no infojson", func(t *testing.T) {
		t.Parallel()
		metadataCopy := make(map[string]any)
		for k, v := range metadata {
			metadataCopy[k] = v
		}
		metadataCopy["infojson_filename"] = nil

		result, err := app.MetadataParserParseForMediaItem(metadataCopy)
		must(t, err)

		metadataPath := result.MetadataFilepath
		if metadataPath != nil {
			t.Errorf("expected metadata_filepath to be nil, got %v", metadataPath)
		}
	})
}

// Helper functions

func contains(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func endsWith(str, suffix string) bool {
	return len(str) >= len(suffix) && str[len(str)-len(suffix):] == suffix
}

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
