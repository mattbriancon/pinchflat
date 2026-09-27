package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestNfoBuilderBuildAndStoreForMediaItem(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)

	t.Run("returns the filepath", func(t *testing.T) {
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "test.nfo")
		metadata := renderParsedMediaMetadata()

		result, err := core.NfoBuilderBuildAndStoreForMediaItem(filepath, metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != filepath {
			t.Errorf("expected %s, got %s", filepath, result)
		}

		if _, err := os.Stat(result); err != nil {
			t.Errorf("file does not exist: %v", err)
		}
	})

	t.Run("builds an NFO file", func(t *testing.T) {
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "test.nfo")
		metadata := renderParsedMediaMetadata()

		result, err := core.NfoBuilderBuildAndStoreForMediaItem(filepath, metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nfo, err := os.ReadFile(result)
		if err != nil {
			t.Fatalf("failed to read NFO file: %v", err)
		}

		nfoStr := string(nfo)
		if !strings.Contains(nfoStr, `<?xml version="1.0" encoding="UTF-8" standalone="yes" ?>`) {
			t.Error("missing XML declaration")
		}

		if !strings.Contains(nfoStr, "<title>"+metadata["title"].(string)+"</title>") {
			t.Error("missing or incorrect title")
		}
	})

	t.Run("escapes invalid characters", func(t *testing.T) {
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "test.nfo")
		metadata := map[string]any{
			"title":       "hello' & <world>",
			"uploader":    "uploader",
			"id":          "id",
			"description": "description",
			"upload_date": "20210101",
		}

		result, err := core.NfoBuilderBuildAndStoreForMediaItem(filepath, metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nfo, err := os.ReadFile(result)
		if err != nil {
			t.Fatalf("failed to read NFO file: %v", err)
		}

		nfoStr := string(nfo)
		if !strings.Contains(nfoStr, "hello&#39; &amp; &lt;world&gt;") {
			t.Error("invalid characters not properly escaped")
		}
	})

	t.Run("uses the season and episode number from the filepath if it can be determined", func(t *testing.T) {
		metadata := map[string]any{
			"title":       "title",
			"uploader":    "uploader",
			"id":          "id",
			"description": "description",
			"upload_date": "20210101",
		}

		filepath := filepath.Join(ta.Config.TmpfileDirectory, "foo/s0123e456.nfo")

		result, err := core.NfoBuilderBuildAndStoreForMediaItem(filepath, metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nfo, err := os.ReadFile(result)
		if err != nil {
			t.Fatalf("failed to read NFO file: %v", err)
		}

		nfoStr := string(nfo)
		if !strings.Contains(nfoStr, "<season>0123</season>") {
			t.Error("missing or incorrect season")
		}

		if !strings.Contains(nfoStr, "<episode>456</episode>") {
			t.Error("missing or incorrect episode")
		}

		os.RemoveAll(filepath)
	})

	t.Run("uses the upload date if the season and episode number can't be determined", func(t *testing.T) {
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "test.nfo")
		metadata := map[string]any{
			"title":       "title",
			"uploader":    "uploader",
			"id":          "id",
			"description": "description",
			"upload_date": "20210101",
		}

		result, err := core.NfoBuilderBuildAndStoreForMediaItem(filepath, metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nfo, err := os.ReadFile(result)
		if err != nil {
			t.Fatalf("failed to read NFO file: %v", err)
		}

		nfoStr := string(nfo)
		if !strings.Contains(nfoStr, "<season>2021</season>") {
			t.Error("missing or incorrect season from upload date")
		}

		if !strings.Contains(nfoStr, "<episode>0101</episode>") {
			t.Error("missing or incorrect episode from upload date")
		}
	})
}

func TestNfoBuilderBuildAndStoreForSource(t *testing.T) {
	t.Parallel()
	ta := coretest.NewApp(t)

	t.Run("returns the filepath", func(t *testing.T) {
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "test.nfo")
		metadata := renderParsedChannelMetadata()

		result, err := core.NfoBuilderBuildAndStoreForSource(filepath, metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != filepath {
			t.Errorf("expected %s, got %s", filepath, result)
		}

		if _, err := os.Stat(result); err != nil {
			t.Errorf("file does not exist: %v", err)
		}
	})

	t.Run("builds an NFO file", func(t *testing.T) {
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "test.nfo")
		metadata := renderParsedChannelMetadata()

		result, err := core.NfoBuilderBuildAndStoreForSource(filepath, metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nfo, err := os.ReadFile(result)
		if err != nil {
			t.Fatalf("failed to read NFO file: %v", err)
		}

		nfoStr := string(nfo)
		if !strings.Contains(nfoStr, `<?xml version="1.0" encoding="UTF-8" standalone="yes" ?>`) {
			t.Error("missing XML declaration")
		}

		if !strings.Contains(nfoStr, "<title>"+metadata["title"].(string)+"</title>") {
			t.Error("missing or incorrect title")
		}
	})

	t.Run("escapes invalid characters", func(t *testing.T) {
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "test.nfo")
		metadata := map[string]any{
			"title":       "hello' & <world>",
			"description": "description",
			"id":          "id",
		}

		result, err := core.NfoBuilderBuildAndStoreForSource(filepath, metadata)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		nfo, err := os.ReadFile(result)
		if err != nil {
			t.Fatalf("failed to read NFO file: %v", err)
		}

		nfoStr := string(nfo)
		if !strings.Contains(nfoStr, "hello&#39; &amp; &lt;world&gt;") {
			t.Error("invalid characters not properly escaped")
		}
	})
}

// Helper functions to create test metadata
func renderParsedMediaMetadata() map[string]any {
	return map[string]any{
		"title":       "Test Video Title",
		"uploader":    "Test Uploader",
		"id":          "test_id_123",
		"description": "Test description for video",
		"upload_date": "20210101",
	}
}

func renderParsedChannelMetadata() map[string]any {
	return map[string]any{
		"title":       "Test Channel",
		"description": "Test channel description",
		"id":          "test_channel_id",
	}
}
