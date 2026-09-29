package app_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

func TestSourceMetadataStorageWorker_KickoffWithTask(t *testing.T) {
	t.Parallel()

	t.Run("enqueues a new worker for the source", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		if _, err := ta.SourceMetadataStorageWorkerKickoffWithTask(ta.Ctx, source); err != nil {
			t.Errorf("SourceMetadataStorageWorkerKickoffWithTask failed: %v", err)
		}

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.SourceMetadataStorageWorkerName, Args: map[string]any{"id": source.ID}})
		if len(enqueued) != 1 {
			t.Errorf("Expected 1 job enqueued, got %d", len(enqueued))
		}
	})

	t.Run("creates a new task for the source", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		task, err := ta.SourceMetadataStorageWorkerKickoffWithTask(ta.Ctx, source)
		if err != nil {
			t.Errorf("SourceMetadataStorageWorkerKickoffWithTask failed: %v", err)
		}

		if task.SourceID == nil || *task.SourceID != source.ID {
			t.Errorf("Expected task.source_id to be %d, got %v", source.ID, task.SourceID)
		}
	})
}

func TestSourceMetadataStorageWorker_Perform(t *testing.T) {
	t.Parallel()

	t.Run("won't call itself in an infinite loop", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		enqueued := ta.Oban.Enqueued(t, obanlite.Match{Worker: app.SourceMetadataStorageWorkerName})
		if len(enqueued) != 0 {
			t.Errorf("Expected 0 jobs enqueued, got %d", len(enqueued))
		}
	})

	t.Run("does not blow up if the record doesn't exist", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		err := ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": int64(0)})
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}

func TestSourceMetadataStorageWorker_PerformAttributeUpdates(t *testing.T) {
	t.Parallel()

	t.Run("the source description is saved", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{Description: store.Ptr("")})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := apptest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		if source.Description != nil {
			t.Errorf("Expected source description to be nil, got %v", source.Description)
		}

		sourceID := source.ID
		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": sourceID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, sourceID)

		if reloadedSource.Description == nil || *reloadedSource.Description != "This is a test file for Pinchflat" {
			t.Errorf("Expected source description to be 'This is a test file for Pinchflat', got %v", reloadedSource.Description)
		}
	})
}

func TestSourceMetadataStorageWorker_PerformMetadataStorage(t *testing.T) {
	t.Parallel()

	t.Run("sets metadata location for source", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)
		reloadedSource, _ = ta.PreloadSourceMetadata(ta.Ctx, reloadedSource)

		if reloadedSource.Metadata == nil || reloadedSource.Metadata.MetadataFilepath == "" {
			t.Errorf("Expected metadata filepath to be set, got %v", reloadedSource.Metadata)
		}

		if reloadedSource.Metadata != nil && reloadedSource.Metadata.MetadataFilepath != "" {
			os.Remove(reloadedSource.Metadata.MetadataFilepath)
		}
	})

	t.Run("fetches and stores returned metadata for source", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})
		fileContents := `{"title": "test"}`

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return fileContents, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)
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
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := apptest.RenderMetadataWithFixedPaths("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)
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

		ta.SourcesDeleteSource(ta.Ctx, reloadedSource, false)
	})

	t.Run("stores metadata images for source", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := apptest.RenderMetadataWithFixedPaths("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)
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

		ta.SourcesDeleteSource(ta.Ctx, reloadedSource, false)
	})
}

func TestSourceMetadataStorageWorker_PerformSourceImageDownloading(t *testing.T) {
	t.Parallel()

	t.Run("downloads and stores source images", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSourceImages: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(profile.ID)})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := apptest.RenderMetadataWithFixedPaths("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)

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

		ta.SourcesDeleteSource(ta.Ctx, reloadedSource, false)
	})

	t.Run("calls appropriate yt-dlp opts by collection type", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name              string
			collectionType    string
			expectedPlaylist  any
			expectedThumbnail string
		}{
			{"channel", "channel", 0, "write_all_thumbnails"},
			{"playlist", "playlist", 1, "write_thumbnail"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ta := apptest.NewApp(t)
				profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSourceImages: store.Ptr(true)})
				source := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(profile.ID), CollectionType: store.Ptr(store.SourceCollectionType(tt.collectionType))})

				ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
					if action == "get_source_details" {
						return apptest.SourceDetailsReturnFixture(map[string]any{
							"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
						}), nil
					}
					if action == "get_source_metadata" {
						if !opts.Contains(ytdlp.Arg{Key: "playlist_items", Value: tt.expectedPlaylist}) {
							t.Errorf("Expected playlist_items=%v for %s", tt.expectedPlaylist, tt.name)
						}
						if !opts.Contains(ytdlp.Arg{Key: tt.expectedThumbnail, Flag: true}) {
							t.Errorf("Expected %s for %s", tt.expectedThumbnail, tt.name)
						}

						metadata, _ := apptest.RenderMetadata("channel_source_metadata")
						return metadata, nil
					}
					return "", nil
				})

				ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})
			})
		}
	})

	t.Run("does not store source images if the profile is not set to", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSourceImages: store.Ptr(false)})
		source := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(profile.ID)})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := apptest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)

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
		t.Parallel()
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSourceImages: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(profile.ID)})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "foo", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				metadata, _ := apptest.RenderMetadata("channel_source_metadata")
				return metadata, nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)

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

	t.Run("sets use_cookies based on source behavior", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name            string
			cookieBehaviour string
			expectedCookies bool
		}{
			{"all_operations", "all_operations", true},
			{"when_needed", "when_needed", false},
			{"disabled", "disabled", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ta := apptest.NewApp(t)
				profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadSourceImages: store.Ptr(true)})
				source := apptest.SourceFixture(t, ta, store.SourceParams{MediaProfileID: store.Ptr(profile.ID), CookieBehaviour: store.Ptr(store.SourceCookieBehaviour(tt.cookieBehaviour))})

				callCount := 0
				ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
					callCount++
					useCookies := addl.UseCookies
					if useCookies != tt.expectedCookies {
						t.Errorf("Expected use_cookies=%v in addl, got %v", tt.expectedCookies, useCookies)
					}
					if action == "get_source_details" {
						return apptest.SourceDetailsReturnFixture(map[string]any{
							"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
						}), nil
					}
					if action == "get_source_metadata" {
						metadata, _ := apptest.RenderMetadata("channel_source_metadata")
						return metadata, nil
					}
					return "", nil
				})

				ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

				if callCount < 2 {
					t.Errorf("Expected at least 2 yt-dlp calls, got %d", callCount)
				}
			})
		}
	})
}

func TestSourceMetadataStorageWorker_PerformSeriesDirectory(t *testing.T) {
	t.Parallel()

	t.Run("sets the series directory based on the returned media filepath", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{SeriesDirectory: store.Ptr("")})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)

		if reloadedSource.SeriesDirectory == nil {
			t.Errorf("Expected series directory to be set")
		}
	})

	t.Run("does not set the series directory if it cannot be determined", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.SourceParams{SeriesDirectory: store.Ptr("")})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "foo", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)

		if reloadedSource.SeriesDirectory != nil {
			t.Errorf("Expected series directory to be nil, got %v", reloadedSource.SeriesDirectory)
		}
	})

	t.Run("sets use_cookies based on source behavior during series directory determination", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name            string
			cookieBehaviour string
			expectedCookies bool
		}{
			{"all_operations", "all_operations", true},
			{"when_needed", "when_needed", false},
			{"disabled", "disabled", false},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				ta := apptest.NewApp(t)
				source := apptest.SourceFixture(t, ta, store.SourceParams{SeriesDirectory: store.Ptr(""), CookieBehaviour: store.Ptr(store.SourceCookieBehaviour(tt.cookieBehaviour))})

				callCount := 0
				ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
					callCount++
					useCookies := addl.UseCookies
					if useCookies != tt.expectedCookies {
						t.Errorf("Expected use_cookies=%v in addl, got %v", tt.expectedCookies, useCookies)
					}
					if action == "get_source_details" {
						return apptest.SourceDetailsReturnFixture(map[string]any{
							"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
						}), nil
					}
					if action == "get_source_metadata" {
						return "{}", nil
					}
					return "", nil
				})

				ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

				if callCount < 2 {
					t.Errorf("Expected at least 2 yt-dlp calls, got %d", callCount)
				}
			})
		}
	})
}

func TestSourceMetadataStorageWorker_PerformNfo(t *testing.T) {
	t.Parallel()

	t.Run("stores the NFO if specified", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadNfo: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.SourceParams{NfoFilepath: store.Ptr(""), MediaProfileID: store.Ptr(profile.ID)})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)

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
		t.Parallel()
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadNfo: store.Ptr(false)})
		source := apptest.SourceFixture(t, ta, store.SourceParams{NfoFilepath: store.Ptr(""), MediaProfileID: store.Ptr(profile.ID)})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "Season 1", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)

		if reloadedSource.NfoFilepath != nil {
			t.Errorf("Expected nfo filepath to be nil, got %v", reloadedSource.NfoFilepath)
		}
	})

	t.Run("does not store the NFO if the series directory cannot be determined", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadNfo: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.SourceParams{NfoFilepath: store.Ptr(""), MediaProfileID: store.Ptr(profile.ID)})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			if action == "get_source_details" {
				return apptest.SourceDetailsReturnFixture(map[string]any{
					"filename": filepath.Join(ta.Config.MediaDirectory, "foo", "bar.mp4"),
				}), nil
			}
			if action == "get_source_metadata" {
				return "{}", nil
			}
			return "", nil
		})

		ta.Oban.PerformJob(ta.Ctx, app.SourceMetadataStorageWorkerName, map[string]any{"id": source.ID})

		reloadedSource, _ := ta.GetSource(ta.Ctx, source.ID)

		if reloadedSource.NfoFilepath != nil {
			t.Errorf("Expected nfo filepath to be nil, got %v", reloadedSource.NfoFilepath)
		}
	})
}
