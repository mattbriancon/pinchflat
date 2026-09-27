package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

func TestSourceMetadataStorageWorker_KickoffWithTask(t *testing.T) {
	t.Run("enqueues a new worker for the source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		if _, err := ta.SourceMetadataStorageWorkerKickoffWithTask(ta.Ctx, source, core.KW{}); err != nil {
			t.Errorf("SourceMetadataStorageWorkerKickoffWithTask failed: %v", err)
		}

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.SourceMetadataStorageWorkerName, Args: map[string]any{"id": source.ID}})
		if len(enqueued) != 1 {
			t.Errorf("Expected 1 job enqueued, got %d", len(enqueued))
		}
	})

	t.Run("creates a new task for the source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		task, err := ta.SourceMetadataStorageWorkerKickoffWithTask(ta.Ctx, source, core.KW{})
		if err != nil {
			t.Errorf("SourceMetadataStorageWorkerKickoffWithTask failed: %v", err)
		}

		if task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("Expected task.source_id to be %d, got %v", source.ID, task.SourceID)
		}
	})
}

func TestSourceMetadataStorageWorker_Perform(t *testing.T) {
	t.Run("won't call itself in an infinite loop", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: core.SourceMetadataStorageWorkerName})
		if len(enqueued) != 0 {
			t.Errorf("Expected 0 jobs enqueued, got %d", len(enqueued))
		}
	})

	t.Run("does not blow up if the record doesn't exist", func(t *testing.T) {
		ta := coretest.NewApp(t)

		err := ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": int64(0)})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestSourceMetadataStorageWorker_PerformAttributeUpdates(t *testing.T) {
	t.Run("the source description is saved", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"description": nil})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := coretest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		if source.Description != nil {
			t.Errorf("Expected source description to be nil, got %v", source.Description)
		}

		sourceID := source.ID
		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": sourceID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, sourceID)

		if reloadedSource.Description == nil || *reloadedSource.Description != "This is a test file for Pinchflat" {
			t.Errorf("Expected source description to be 'This is a test file for Pinchflat', got %v", reloadedSource.Description)
		}
	})
}

func TestSourceMetadataStorageWorker_PerformMetadataStorage(t *testing.T) {
	t.Run("sets metadata location for source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)
		reloadedSource, _ = ta.PreloadSourceMetadata(ta.Ctx, reloadedSource)

		if reloadedSource.Metadata == nil || reloadedSource.Metadata.MetadataFilepath == "" {
			t.Errorf("Expected metadata filepath to be set, got %v", reloadedSource.Metadata)
		}

		if reloadedSource.Metadata != nil && reloadedSource.Metadata.MetadataFilepath != "" {
			os.Remove(reloadedSource.Metadata.MetadataFilepath)
		}
	})

	t.Run("fetches and stores returned metadata for source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})
		fileContents := `{"title": "test"}`

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return fileContents, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)
		reloadedSource, _ = ta.PreloadSourceMetadata(ta.Ctx, reloadedSource)

		metadata, err := ta.MetadataFileHelpersReadCompressedMetadata(ta.Ctx, reloadedSource.Metadata.MetadataFilepath)
		if err != nil {
			t.Errorf("Failed to read metadata: %v", err)
		}

		if metadata == nil || metadata["title"] != "test" {
			t.Errorf("Expected metadata title to be 'test', got %v", metadata)
		}
	})

	t.Run("sets metadata image location for source", func(t *testing.T) {

		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := coretest.RenderMetadataWithFixedPaths("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)
		reloadedSource, _ = ta.PreloadSourceMetadata(ta.Ctx, reloadedSource)

		if reloadedSource.Metadata.FanartFilepath == nil || *reloadedSource.Metadata.FanartFilepath == "" {
			t.Errorf("Expected fanart_filepath to be set")
		}
		if reloadedSource.Metadata.PosterFilepath == nil || *reloadedSource.Metadata.PosterFilepath == "" {
			t.Errorf("Expected poster_filepath to be set")
		}
		if reloadedSource.Metadata.BannerFilepath == nil || *reloadedSource.Metadata.BannerFilepath == "" {
			t.Errorf("Expected banner_filepath to be set")
		}

		ta.SourcesDeleteSource(ta.Ctx, reloadedSource, core.KW{core.Flag("delete_files")})
	})

	t.Run("stores metadata images for source", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := coretest.RenderMetadataWithFixedPaths("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)
		reloadedSource, _ = ta.PreloadSourceMetadata(ta.Ctx, reloadedSource)

		if reloadedSource.Metadata.FanartFilepath == nil {
			t.Error("Expected fanart_filepath to be set")
		} else if _, err := os.Stat(*reloadedSource.Metadata.FanartFilepath); err != nil {
			t.Errorf("Expected fanart file to exist at %s", *reloadedSource.Metadata.FanartFilepath)
		}
		if reloadedSource.Metadata.PosterFilepath == nil {
			t.Error("Expected poster_filepath to be set")
		} else if _, err := os.Stat(*reloadedSource.Metadata.PosterFilepath); err != nil {
			t.Errorf("Expected poster file to exist at %s", *reloadedSource.Metadata.PosterFilepath)
		}
		if reloadedSource.Metadata.BannerFilepath == nil {
			t.Error("Expected banner_filepath to be set")
		} else if _, err := os.Stat(*reloadedSource.Metadata.BannerFilepath); err != nil {
			t.Errorf("Expected banner file to exist at %s", *reloadedSource.Metadata.BannerFilepath)
		}

		ta.SourcesDeleteSource(ta.Ctx, reloadedSource, core.KW{core.Flag("delete_files")})
	})
}

func TestSourceMetadataStorageWorker_PerformSourceImageDownloading(t *testing.T) {
	t.Run("downloads and stores source images", func(t *testing.T) {

		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_source_images": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": profile.ID})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := coretest.RenderMetadataWithFixedPaths("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)

		if reloadedSource.FanartFilepath == nil || *reloadedSource.FanartFilepath == "" {
			t.Errorf("Expected fanart_filepath to be set")
		}
		if reloadedSource.PosterFilepath == nil || *reloadedSource.PosterFilepath == "" {
			t.Errorf("Expected poster_filepath to be set")
		}
		if reloadedSource.BannerFilepath == nil || *reloadedSource.BannerFilepath == "" {
			t.Errorf("Expected banner_filepath to be set")
		}

		if reloadedSource.FanartFilepath != nil {
			if _, err := os.Stat(*reloadedSource.FanartFilepath); err != nil {
				t.Errorf("Expected fanart file to exist")
			}
		}
		if reloadedSource.PosterFilepath != nil {
			if _, err := os.Stat(*reloadedSource.PosterFilepath); err != nil {
				t.Errorf("Expected poster file to exist")
			}
		}
		if reloadedSource.BannerFilepath != nil {
			if _, err := os.Stat(*reloadedSource.BannerFilepath); err != nil {
				t.Errorf("Expected banner file to exist")
			}
		}

		ta.SourcesDeleteSource(ta.Ctx, reloadedSource, core.KW{core.Flag("delete_files")})
	})

	t.Run("calls one set of yt-dlp metadata opts for channels", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_source_images": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": profile.ID, "collection_type": "channel"})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				// Check that the expected options are present
				if !opts.Contains(core.Opt("playlist_items", 0)) {
					t.Errorf("Expected playlist_items=0 for channel")
				}
				if !opts.HasFlag("write_all_thumbnails") {
					t.Errorf("Expected write_all_thumbnails for channel")
				}

				metadata, _ := coretest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})
	})

	t.Run("calls another set of yt-dlp metadata opts for playlists", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_source_images": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": profile.ID, "collection_type": "playlist"})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				// Check that the expected options are present for playlists
				if !opts.Contains(core.Opt("playlist_items", 1)) {
					t.Errorf("Expected playlist_items=1 for playlist")
				}
				if !opts.HasFlag("write_thumbnail") {
					t.Errorf("Expected write_thumbnail for playlist")
				}

				metadata, _ := coretest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})
	})

	t.Run("does not store source images if the profile is not set to", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_source_images": false})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": profile.ID})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := coretest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)

		if reloadedSource.FanartFilepath != nil {
			t.Errorf("Expected fanart_filepath to be nil, got %v", reloadedSource.FanartFilepath)
		}
		if reloadedSource.PosterFilepath != nil {
			t.Errorf("Expected poster_filepath to be nil, got %v", reloadedSource.PosterFilepath)
		}
		if reloadedSource.BannerFilepath != nil {
			t.Errorf("Expected banner_filepath to be nil, got %v", reloadedSource.BannerFilepath)
		}
	})

	t.Run("does not store source images if the series directory cannot be determined", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_source_images": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": profile.ID})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "foo", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := coretest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)

		if reloadedSource.FanartFilepath != nil {
			t.Errorf("Expected fanart_filepath to be nil")
		}
		if reloadedSource.PosterFilepath != nil {
			t.Errorf("Expected poster_filepath to be nil")
		}
		if reloadedSource.BannerFilepath != nil {
			t.Errorf("Expected banner_filepath to be nil")
		}
	})

	t.Run("sets use_cookies if the source uses cookies", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_source_images": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": profile.ID, "cookie_behaviour": "all_operations"})

		callCount := 0
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			callCount++
			useCookies, _ := addl.Get("use_cookies")
			if useCookies != true {
				t.Errorf("Expected use_cookies=true in addl, got %v", useCookies)
			}
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := coretest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		if callCount < 2 {
			t.Errorf("Expected at least 2 yt-dlp calls, got %d", callCount)
		}
	})

	t.Run("does not set use_cookies if the source uses cookies when needed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_source_images": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": profile.ID, "cookie_behaviour": "when_needed"})

		callCount := 0
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			callCount++
			useCookies, _ := addl.Get("use_cookies")
			if useCookies != false {
				t.Errorf("Expected use_cookies=false in addl, got %v", useCookies)
			}
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := coretest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		if callCount < 2 {
			t.Errorf("Expected at least 2 yt-dlp calls, got %d", callCount)
		}
	})

	t.Run("does not set use_cookies if the source does not use cookies", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_source_images": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"media_profile_id": profile.ID, "cookie_behaviour": "disabled"})

		callCount := 0
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			callCount++
			useCookies, _ := addl.Get("use_cookies")
			if useCookies != false {
				t.Errorf("Expected use_cookies=false in addl, got %v", useCookies)
			}
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := coretest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		if callCount < 2 {
			t.Errorf("Expected at least 2 yt-dlp calls, got %d", callCount)
		}
	})
}

func TestSourceMetadataStorageWorker_PerformSeriesDirectory(t *testing.T) {
	t.Run("sets the series directory based on the returned media filepath", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"series_directory": nil})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)

		if reloadedSource.SeriesDirectory == nil {
			t.Errorf("Expected series directory to be set")
		}
	})

	t.Run("does not set the series directory if it cannot be determined", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"series_directory": nil})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "foo", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)

		if reloadedSource.SeriesDirectory != nil {
			t.Errorf("Expected series directory to be nil, got %v", reloadedSource.SeriesDirectory)
		}
	})

	t.Run("sets use_cookies if the source is set to use cookies", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"series_directory": nil, "cookie_behaviour": "all_operations"})

		callCount := 0
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			callCount++
			useCookies, _ := addl.Get("use_cookies")
			if useCookies != true {
				t.Errorf("Expected use_cookies=true in addl, got %v", useCookies)
			}
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		if callCount < 2 {
			t.Errorf("Expected at least 2 yt-dlp calls, got %d", callCount)
		}
	})

	t.Run("does not set use_cookies if the source uses cookies when needed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"series_directory": nil, "cookie_behaviour": "when_needed"})

		callCount := 0
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			callCount++
			useCookies, _ := addl.Get("use_cookies")
			if useCookies != false {
				t.Errorf("Expected use_cookies=false in addl, got %v", useCookies)
			}
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		if callCount < 2 {
			t.Errorf("Expected at least 2 yt-dlp calls, got %d", callCount)
		}
	})

	t.Run("does not set use_cookies if the source is not set to use cookies", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := coretest.SourceFixture(t, ta, core.Attrs{"series_directory": nil, "cookie_behaviour": "disabled"})

		callCount := 0
		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			callCount++
			useCookies, _ := addl.Get("use_cookies")
			if useCookies != false {
				t.Errorf("Expected use_cookies=false in addl, got %v", useCookies)
			}
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		if callCount < 2 {
			t.Errorf("Expected at least 2 yt-dlp calls, got %d", callCount)
		}
	})
}

func TestSourceMetadataStorageWorker_PerformNfo(t *testing.T) {
	t.Run("stores the NFO if specified", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_nfo": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"nfo_filepath": nil, "media_profile_id": profile.ID})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)

		if reloadedSource.NfoFilepath == nil || *reloadedSource.NfoFilepath == "" {
			t.Errorf("Expected nfo filepath to be set")
		}

		expectedNfoPath := filepath.Join(*reloadedSource.SeriesDirectory, "tvshow.nfo")
		if reloadedSource.NfoFilepath != nil && *reloadedSource.NfoFilepath != expectedNfoPath {
			t.Errorf("Expected nfo filepath to be %s, got %s", expectedNfoPath, *reloadedSource.NfoFilepath)
		}

		if reloadedSource.NfoFilepath != nil && *reloadedSource.NfoFilepath != "" {
			os.Remove(*reloadedSource.NfoFilepath)
		}
	})

	t.Run("does not store the NFO if not specified", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_nfo": false})
		source := coretest.SourceFixture(t, ta, core.Attrs{"nfo_filepath": nil, "media_profile_id": profile.ID})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)

		if reloadedSource.NfoFilepath != nil {
			t.Errorf("Expected nfo filepath to be nil, got %v", reloadedSource.NfoFilepath)
		}
	})

	t.Run("does not store the NFO if the series directory cannot be determined", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{"download_nfo": true})
		source := coretest.SourceFixture(t, ta, core.Attrs{"nfo_filepath": nil, "media_profile_id": profile.ID})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action == "get_source_details" {
				return coretest.SourceDetailsReturnFixture(core.Attrs{
					"filename": filepath.Join(ta.Config.MediaDirectory, "foo", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, core.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.SourcesGetSource(ta.Ctx, source.ID)

		if reloadedSource.NfoFilepath != nil {
			t.Errorf("Expected nfo filepath to be nil, got %v", reloadedSource.NfoFilepath)
		}
	})
}
