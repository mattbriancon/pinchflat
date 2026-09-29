package app_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
)

func TestSourceImageParserStoreSourceImages(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)

	t.Run("returns a map of image types and locations", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		metadata := renderParsedChannelSourceMetadata()

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if len(result) == 0 {
			t.Error("expected result map to contain entries")
		}

		// Verify expected keys exist
		for _, key := range []string{"banner_filepath", "fanart_filepath", "poster_filepath"} {
			if _, exists := result[key]; exists {
				// Verify file exists
				if filepath, exists := result[key]; exists {
					if _, err := os.Stat(filepath); err != nil {
						t.Errorf("file does not exist for %s: %v", key, err)
					}
				}
			}
		}
	})

	t.Run("returns the avatar_uncropped as the poster", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		sourceImagePath := apptest.RepoPath("testdata/support/files/channel_photos/a.0.jpg")
		metadata := map[string]any{
			"thumbnails": []any{
				map[string]any{
					"id":       "avatar_uncropped",
					"filepath": sourceImagePath,
				},
			},
		}

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if _, exists := result["poster_filepath"]; !exists {
			t.Error("expected poster_filepath in result")
		}

		if len(result) != 1 {
			t.Errorf("expected only poster, got %d items", len(result))
		}
	})

	t.Run("returns the banner_uncropped as the fanart", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		sourceImagePath := apptest.RepoPath("testdata/support/files/channel_photos/a.0.jpg")
		metadata := map[string]any{
			"thumbnails": []any{
				map[string]any{
					"id":       "banner_uncropped",
					"filepath": sourceImagePath,
				},
			},
		}

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if _, exists := result["fanart_filepath"]; !exists {
			t.Error("expected fanart_filepath in result")
		}

		if len(result) != 1 {
			t.Errorf("expected only fanart, got %d items", len(result))
		}
	})

	t.Run("doesn't return a banner if no suitable images are found", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		metadata := map[string]any{
			"thumbnails": []any{
				map[string]any{
					"id":       "1",
					"filepath": "foo.jpg",
					"width":    float64(100),
					"height":   float64(100),
				},
			},
		}

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})

	t.Run("doesn't blow up if empty metadata is passed", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		metadata := map[string]any{}

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if len(result) != 0 {
			t.Errorf("expected empty result for empty metadata, got %d items", len(result))
		}
	})

	t.Run("doesn't blow up if no thumbnails are passed", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		metadata := map[string]any{
			"thumbnails": []any{},
		}

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if len(result) != 0 {
			t.Errorf("expected empty result for no thumbnails, got %d items", len(result))
		}
	})
}

func TestSourceImageParserStoreSourceImagesFallbacks(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)

	t.Run("uses the entries list for a fallback poster if needed", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		sourceImagePath := apptest.RepoPath("testdata/support/files/channel_photos/a.0.jpg")
		metadata := map[string]any{
			"thumbnails": []any{},
			"entries": []any{
				map[string]any{
					"thumbnails": []any{
						map[string]any{
							"filepath": sourceImagePath,
						},
					},
				},
			},
		}

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if _, exists := result["poster_filepath"]; !exists {
			t.Error("expected poster_filepath in result from fallback")
		}
	})

	t.Run("doesn't blow up if the entries list doesn't have any suitable thumbnails", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		metadata := map[string]any{
			"thumbnails": []any{},
			"entries": []any{
				map[string]any{
					"thumbnails": []any{
						map[string]any{
							"id": "1",
						},
					},
				},
			},
		}

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})

	t.Run("doesn't use the entries list if it's empty", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		metadata := map[string]any{
			"thumbnails": []any{},
			"entries":    []any{},
		}

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})

	t.Run("doesn't use the entries list if it's not present", func(t *testing.T) {
		baseDir := ta.Config.TmpfileDirectory
		metadata := map[string]any{
			"thumbnails": []any{},
		}

		result, err := app.SourceImageParserStoreSourceImages(baseDir, metadata)
		must(t, err)

		if len(result) != 0 {
			t.Errorf("expected empty result, got %d items", len(result))
		}
	})
}

// Helper function to create parsed channel source metadata
func renderParsedChannelSourceMetadata() map[string]any {
	sourceImagePath := apptest.RepoPath("testdata/support/files/channel_photos/a.0.jpg")
	return map[string]any{
		"thumbnails": []any{
			map[string]any{
				"id":       "avatar_uncropped",
				"filepath": sourceImagePath,
			},
			map[string]any{
				"id":       "banner_uncropped",
				"filepath": sourceImagePath,
			},
			map[string]any{
				"id":       "1",
				"filepath": sourceImagePath,
				"width":    float64(1000),
				"height":   float64(100),
			},
		},
	}
}
