package app_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestMetadataFileHelpers_MetadataDirectoryFor(t *testing.T) {
	t.Parallel()
	t.Run("returns the metadata directory for the given record", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, err := ta.App.PreloadMediaItemSource(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to preload source: %v", err)
		}

		metadataDirectory, err := ta.App.MetadataFileHelpersMetadataDirectoryFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to get metadata directory: %v", err)
		}

		baseMetadataDirectory := ta.App.Config.MetadataDirectory
		expectedDirectory := filepath.Join(baseMetadataDirectory, "media_items", "1")

		if metadataDirectory != expectedDirectory {
			t.Errorf("expected %q, got %q", expectedDirectory, metadataDirectory)
		}
	})
}

func TestMetadataFileHelpers_CompressAndStoreMetadataFor(t *testing.T) {
	t.Parallel()
	t.Run("returns the filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		metadataMap := map[string]any{"foo": "bar"}

		filePath, err := ta.App.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, mediaItem, metadataMap)
		if err != nil {
			t.Fatalf("failed to compress and store metadata: %v", err)
		}

		// Should end with /media_items/{id}/metadata.json.gz
		if !bytes.Contains([]byte(filePath), []byte("/media_items/")) || !bytes.Contains([]byte(filePath), []byte("/metadata.json.gz")) {
			t.Errorf("expected filepath to contain /media_items/ and /metadata.json.gz, got %q", filePath)
		}
	})

	t.Run("creates folder structure based on passed record", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		metadataMap := map[string]any{"foo": "bar"}

		filePath, err := ta.App.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, mediaItem, metadataMap)
		if err != nil {
			t.Fatalf("failed to compress and store metadata: %v", err)
		}

		// Check that the directory exists
		dir := filepath.Dir(filePath)
		if !dirExists(dir) {
			t.Errorf("expected directory %q to exist", dir)
		}
	})

	t.Run("stores it as compressed JSON", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		metadataMap := map[string]any{"foo": "bar"}

		filePath, err := ta.App.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, mediaItem, metadataMap)
		if err != nil {
			t.Fatalf("failed to compress and store metadata: %v", err)
		}

		// Read and decompress the file
		file, err := os.Open(filePath)
		if err != nil {
			t.Fatalf("failed to open file: %v", err)
		}
		defer file.Close()

		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			t.Fatalf("failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, gzipReader); err != nil {
			t.Fatalf("failed to read gzipped content: %v", err)
		}

		expectedJSON, err := db.EncodeJSON(metadataMap)
		if err != nil {
			t.Fatalf("failed to encode JSON: %v", err)
		}

		if buf.String() != expectedJSON {
			t.Errorf("expected %q, got %q", expectedJSON, buf.String())
		}
	})
}

func TestMetadataFileHelpers_ReadCompressedMetadata(t *testing.T) {
	t.Parallel()
	t.Run("returns the compressed and decoded metadata", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		metadataMap := map[string]any{"foo": "bar"}

		filePath, err := ta.App.MetadataFileHelpersCompressAndStoreMetadataFor(ta.Ctx, mediaItem, metadataMap)
		if err != nil {
			t.Fatalf("failed to compress and store metadata: %v", err)
		}

		decodedJSON, err := ta.App.MetadataFileHelpersReadCompressedMetadata(ta.Ctx, filePath)
		if err != nil {
			t.Fatalf("failed to read compressed metadata: %v", err)
		}

		if decodedJSON["foo"] != "bar" {
			t.Errorf("expected foo=\"bar\", got %v", decodedJSON)
		}
	})
}

func TestMetadataFileHelpers_DownloadAndStoreThumbnailFor(t *testing.T) {
	t.Parallel()
	t.Run("returns the filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, err := ta.App.PreloadMediaItemSource(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to preload source: %v", err)
		}

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", nil
		})

		result, err := ta.App.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to download and store thumbnail: %v", err)
		}

		if result == nil {
			t.Errorf("expected filepath, got nil")
		} else if !bytes.Contains([]byte(*result), []byte("/media_items/")) || !bytes.Contains([]byte(*result), []byte("/thumbnail.jpg")) {
			t.Errorf("expected filepath to contain /media_items/ and /thumbnail.jpg, got %q", *result)
		}
	})

	t.Run("calls yt-dlp with the expected options", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, err := ta.App.PreloadMediaItemSource(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to preload source: %v", err)
		}

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			if url != mediaItem.OriginalURL {
				t.Errorf("expected url %q, got %q", mediaItem.OriginalURL, url)
			}
			if action != "download_thumbnail" {
				t.Errorf("expected action \"download_thumbnail\", got %q", action)
			}
			if ot != "after_move:%()j" {
				t.Errorf("expected output template \"after_move:%%()j\", got %q", ot)
			}

			// Check that opts includes expected flags and settings
			// Expected: no_simulate, skip_download, write_thumbnail, convert_thumbnail: jpg, output: path
			hasNoSimulate := false
			hasSkipDownload := false
			hasWriteThumbnail := false
			hasConvertThumbnail := false
			var outputPath string

			for _, kv := range opts {
				if kv.Key == "no_simulate" && kv.Flag {
					hasNoSimulate = true
				} else if kv.Key == "skip_download" && kv.Flag {
					hasSkipDownload = true
				} else if kv.Key == "write_thumbnail" && kv.Flag {
					hasWriteThumbnail = true
				} else if kv.Key == "convert_thumbnail" && kv.Value == "jpg" {
					hasConvertThumbnail = true
				} else if kv.Key == "output" {
					outputPath = kv.Value.(string)
				}
			}

			if !hasNoSimulate {
				t.Errorf("expected no_simulate flag in opts")
			}
			if !hasSkipDownload {
				t.Errorf("expected skip_download flag in opts")
			}
			if !hasWriteThumbnail {
				t.Errorf("expected write_thumbnail flag in opts")
			}
			if !hasConvertThumbnail {
				t.Errorf("expected convert_thumbnail: jpg in opts")
			}
			if !bytes.Contains([]byte(outputPath), []byte("/media_items/")) || !bytes.Contains([]byte(outputPath), []byte("/thumbnail.")) {
				t.Errorf("expected output path to contain /media_items/ and /thumbnail., got %q", outputPath)
			}

			return "", nil
		})

		_, err = ta.App.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to download and store thumbnail: %v", err)
		}
	})

	t.Run("returns nil if yt-dlp fails", func(t *testing.T) {
		ta := apptest.NewApp(t)

		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, err := ta.App.PreloadMediaItemSource(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to preload source: %v", err)
		}

		ta.YtDlpMock.Run.Stub(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			return "", fmt.Errorf("yt-dlp failed")
		})

		result, err := ta.App.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result, got %v", result)
		}
	})
}

func TestMetadataFileHelpers_DownloadAndStoreThumbnailForCookieUsage(t *testing.T) {
	t.Parallel()
	t.Run("sets use_cookies if the source uses cookies", func(t *testing.T) {
		ta := apptest.NewApp(t)

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			// Check that use_cookies is true in addl opts
			found := false
			for _, kv := range addl {
				if kv.Key == "use_cookies" && kv.Value == true {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected use_cookies: true in addl opts, got %v", addl)
			}
			return "", nil
		})

		source := apptest.SourceFixture(t, ta, store.Attrs{
			"cookie_behaviour": store.SourceCookieBehaviourAllOperations,
		})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
		mediaItem, err := ta.App.PreloadMediaItemSource(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to preload source: %v", err)
		}

		_, err = ta.App.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to download and store thumbnail: %v", err)
		}
	})

	t.Run("does not set use_cookies if the source uses cookies when needed", func(t *testing.T) {
		ta := apptest.NewApp(t)

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			// Check that use_cookies is false in addl opts
			found := false
			for _, kv := range addl {
				if kv.Key == "use_cookies" && kv.Value == false {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected use_cookies: false in addl opts, got %v", addl)
			}
			return "", nil
		})

		source := apptest.SourceFixture(t, ta, store.Attrs{
			"cookie_behaviour": store.SourceCookieBehaviourWhenNeeded,
		})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
		mediaItem, err := ta.App.PreloadMediaItemSource(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to preload source: %v", err)
		}

		_, err = ta.App.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to download and store thumbnail: %v", err)
		}
	})

	t.Run("does not set use_cookies if the source does not use cookies", func(t *testing.T) {
		ta := apptest.NewApp(t)

		ta.YtDlpMock.Run.Expect(func(url, action string, opts store.KW, ot string, addl store.KW) (string, error) {
			// Check that use_cookies is false in addl opts
			found := false
			for _, kv := range addl {
				if kv.Key == "use_cookies" && kv.Value == false {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected use_cookies: false in addl opts, got %v", addl)
			}
			return "", nil
		})

		source := apptest.SourceFixture(t, ta, store.Attrs{
			"cookie_behaviour": store.SourceCookieBehaviourDisabled,
		})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
		mediaItem, err := ta.App.PreloadMediaItemSource(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to preload source: %v", err)
		}

		_, err = ta.App.MetadataFileHelpersDownloadAndStoreThumbnailFor(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("failed to download and store thumbnail: %v", err)
		}
	})
}

func TestMetadataFileHelpers_ParseUploadDate(t *testing.T) {
	t.Parallel()
	t.Run("returns a datetime from the given metadata upload date", func(t *testing.T) {
		uploadDate := "20210101"

		result, err := app.MetadataFileHelpersParseUploadDate(uploadDate)
		if err != nil {
			t.Fatalf("failed to parse upload date: %v", err)
		}

		// Expected: 2021-01-01 00:00:00 UTC
		if result.Year() != 2021 || result.Month() != 1 || result.Day() != 1 {
			t.Errorf("expected 2021-01-01, got %d-%02d-%02d", result.Year(), result.Month(), result.Day())
		}
		if result.Hour() != 0 || result.Minute() != 0 || result.Second() != 0 {
			t.Errorf("expected 00:00:00, got %02d:%02d:%02d", result.Hour(), result.Minute(), result.Second())
		}
	})
}

func TestMetadataFileHelpers_SeriesDirectoryFromMediaFilepath(t *testing.T) {
	t.Parallel()
	t.Run("returns base series directory if filepaths are setup as expected", func(t *testing.T) {
		goodFilepaths := []string{
			"/media/season1/episode.mp4",
			"/media/season 1/episode.mp4",
			"/media/season.1/episode.mp4",
			"/media/season_1/episode.mp4",
			"/media/season-1/episode.mp4",
			"/media/SEASON 1/episode.mp4",
			"/media/SEASON.1/episode.mp4",
			"/media/s1/episode.mp4",
			"/media/s.1/episode.mp4",
			"/media/s_1/episode.mp4",
			"/media/s-1/episode.mp4",
			"/media/s 1/episode.mp4",
			"/media/S1/episode.mp4",
			"/media/S.1/episode.mp4",
		}

		for _, filePath := range goodFilepaths {
			t.Run("filepath_"+filePath, func(t *testing.T) {
				result, err := app.MetadataFileHelpersSeriesDirectoryFromMediaFilepath(filePath)
				if err != nil {
					t.Errorf("expected no error for %q, got %v", filePath, err)
				}
				if result != "/media" {
					t.Errorf("expected \"/media\" for %q, got %q", filePath, result)
				}
			})
		}
	})

	t.Run("returns an error if the season filepath can't be determined", func(t *testing.T) {
		badFilepaths := []string{
			"/media/1/episode.mp4",
			"/media/(s1)/episode.mp4",
			"/media/episode.mp4",
			"/media/s1e1/episode.mp4",
			"/media/s1 e1/episode.mp4",
			"/media/s1 (something else)/episode.mp4",
			"/media/season1e1/episode.mp4",
			"/media/season1 e1/episode.mp4",
			"/media/seasoning1/episode.mp4",
			"/media/season/episode.mp4",
			"/media/series1/episode.mp4",
			"/media/s/episode.mp4",
			"/media/foo",
			"/media/bar/",
		}

		for _, filePath := range badFilepaths {
			t.Run("filepath_"+filePath, func(t *testing.T) {
				result, err := app.MetadataFileHelpersSeriesDirectoryFromMediaFilepath(filePath)
				if err == nil {
					t.Errorf("expected error for %q, got result %q", filePath, result)
				}
			})
		}
	})
}

func TestMetadataFileHelpers_SeasonAndEpisodeFromMediaFilepath(t *testing.T) {
	t.Parallel()
	t.Run("returns a season and episode if one can be determined", func(t *testing.T) {
		testCases := []struct {
			filepath string
			season   string
			episode  string
		}{
			{"/foo/s1e2 - test.mp4", "1", "2"},
			{"/foo/S1E2 - test.mp4", "1", "2"},
			{"/foo/s001e002 - test.mp4", "001", "002"},
			{"/foo/s1e2bar - test.mp4", "1", "2"},
			{"/foo/bar s1e2 - test.mp4", "1", "2"},
		}

		for _, tc := range testCases {
			t.Run("filepath_"+tc.filepath, func(t *testing.T) {
				season, episode, err := app.MetadataFileHelpersSeasonAndEpisodeFromMediaFilepath(tc.filepath)
				if err != nil {
					t.Errorf("expected no error for %q, got %v", tc.filepath, err)
				}
				if season != tc.season {
					t.Errorf("expected season %q, got %q", tc.season, season)
				}
				if episode != tc.episode {
					t.Errorf("expected episode %q, got %q", tc.episode, episode)
				}
			})
		}
	})

	t.Run("returns an error if a season and episode can't be determined", func(t *testing.T) {
		badFilepaths := []string{
			"/foo/test.mp4",
			"/foo/s1 - test.mp4",
			"/foo/s1e - test.mp4",
			"/foo/s1etest.mp4",
		}

		for _, filePath := range badFilepaths {
			t.Run("filepath_"+filePath, func(t *testing.T) {
				season, episode, err := app.MetadataFileHelpersSeasonAndEpisodeFromMediaFilepath(filePath)
				if err == nil {
					t.Errorf("expected error for %q, got season=%q, episode=%q", filePath, season, episode)
				}
			})
		}
	})
}

// Helper function to check if directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
