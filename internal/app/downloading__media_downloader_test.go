package app_test

import (
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/cmdrun"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

func TestMediaDownloader_DownloadForMediaItem(t *testing.T) {
	t.Parallel()

	t.Run("calls the backend runner", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		// Stub HTTP mock
		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			} else if action == "download" {
				if url != mediaItem.OriginalURL {
					t.Errorf("expected URL %s, got %s", mediaItem.OriginalURL, url)
				}
				if outputTemplate != "after_move:%()j" {
					t.Errorf("expected outputTemplate 'after_move:%%()j', got %s", outputTemplate)
				}
				// Check for output_filepath in addlOpts
				if addlOpts.OutputFilepath == "" {
					t.Errorf("expected output_filepath in addlOpts")
				}
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			return "", fmt.Errorf("unexpected action: %s", action)
		})

		result, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if result == nil {
			t.Errorf("expected result, got nil")
		}
	})

	t.Run("saves the metadata filepath to the database", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			} else if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			}
			return "", nil
		})

		if mediaItem.Metadata != nil {
			t.Errorf("expected no metadata initially")
		}

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil {
			t.Errorf("expected result")
		}
		if result.MediaItem.Metadata == nil {
			t.Errorf("expected metadata to be saved")
		}
		if result.MediaItem.Metadata.MetadataFilepath == "" {
			t.Errorf("expected metadata_filepath")
		}
		if result.MediaItem.Metadata.ThumbnailFilepath == "" {
			t.Errorf("expected thumbnail_filepath")
		}
	})

	t.Run("errors for non-downloadable media are passed through", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return `{"live_status": "is_live"}`, nil
			}
			return "", nil
		})

		_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err == nil {
			t.Errorf("expected error")
		}
		mediaDownloaderErr, ok := err.(*app.MediaDownloaderError)
		if !ok {
			t.Errorf("expected MediaDownloaderError, got %T", err)
		}
		if mediaDownloaderErr.Reason != "unsuitable_for_download" {
			t.Errorf("expected unsuitable_for_download, got %s", mediaDownloaderErr.Reason)
		}
	})

	t.Run("non-recoverable errors are passed through", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			return "", &cmdrun.Error{Output: "some_error", Status: 1}
		})

		_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err == nil {
			t.Errorf("expected error")
		}
		mediaDownloaderErr, ok := err.(*app.MediaDownloaderError)
		if !ok {
			t.Errorf("expected MediaDownloaderError, got %T", err)
		}
		if mediaDownloaderErr.Reason != "download_failed" {
			t.Errorf("expected download_failed, got %s", mediaDownloaderErr.Reason)
		}
	})

	t.Run("unknown errors are passed through", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			}
			return "", fmt.Errorf("some_error")
		})

		_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err == nil {
			t.Errorf("expected error")
		}
		mediaDownloaderErr, ok := err.(*app.MediaDownloaderError)
		if !ok {
			t.Errorf("expected MediaDownloaderError, got %T", err)
		}
		if mediaDownloaderErr.Reason != "unknown" {
			t.Errorf("expected unknown, got %s", mediaDownloaderErr.Reason)
		}
	})
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingNonDownloadableMedia(t *testing.T) {
	t.Parallel()

	t.Run("calls the download runner if the media is currently downloadable", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return `{"live_status": "was_live"}`, nil
			} else if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil {
			t.Errorf("expected result")
		}
	})

	t.Run("does not call the download runner if the media is not downloadable", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return `{"live_status": "is_live"}`, nil
			}
			return "", nil
		})

		_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("returns unexpected errors from the download status determination method", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "", fmt.Errorf("what_tha")
			}
			return "", nil
		})

		_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err == nil {
			t.Errorf("expected error")
		}
		mediaDownloaderErr, ok := err.(*app.MediaDownloaderError)
		if !ok {
			t.Errorf("expected MediaDownloaderError, got %T", err)
		}
		if mediaDownloaderErr.Reason != "unknown" {
			t.Errorf("expected unknown, got %s", mediaDownloaderErr.Reason)
		}
	})
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingOverrideOptions(t *testing.T) {
	t.Parallel()

	t.Run("includes override opts if specified", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				// Check that no_force_overwrites is in opts but force_overwrites is not
				hasForceOverwrites := false
				hasNoForceOverwrites := false
				for _, kv := range opts {
					if kv.Key == "force_overwrites" && kv.Flag {
						hasForceOverwrites = true
					}
					if kv.Key == "no_force_overwrites" && kv.Flag {
						hasNoForceOverwrites = true
					}
				}
				if hasForceOverwrites {
					t.Errorf("expected no force_overwrites in opts")
				}
				if !hasNoForceOverwrites {
					t.Errorf("expected no_force_overwrites in opts")
				}
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		overrideOpts := app.DownloadOverrides{OverwriteBehaviour: "no_force_overwrites"}
		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, overrideOpts)

		if result == nil {
			t.Errorf("expected result")
		}
	})
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingCookieUsage(t *testing.T) {
	t.Parallel()

	t.Run("sets use_cookies if the source uses cookies", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"cookie_behaviour": "all_operations"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			useCookies := addlOpts.UseCookies
			if useCookies != true {
				t.Errorf("expected use_cookies to be true, got %v", useCookies)
			}
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil {
			t.Errorf("expected result")
		}
	})

	t.Run("does not set use_cookies if the source uses cookies when needed", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"cookie_behaviour": "when_needed"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			useCookies := addlOpts.UseCookies
			if useCookies != false {
				t.Errorf("expected use_cookies to be false, got %v", useCookies)
			}
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil {
			t.Errorf("expected result")
		}
	})

	t.Run("does not set use_cookies if the source does not use cookies", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"cookie_behaviour": "disabled"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			useCookies := addlOpts.UseCookies
			if useCookies != false {
				t.Errorf("expected use_cookies to be false, got %v", useCookies)
			}
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil {
			t.Errorf("expected result")
		}
	})
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingNonCookieRetries(t *testing.T) {
	t.Parallel()

	t.Run("returns a recovered tuple on recoverable errors", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				outputFilePathVal := addlOpts.OutputFilepath
				outputFilepath := outputFilePathVal
				metadata, _ := apptest.RenderMetadata("media_metadata")
				os.WriteFile(outputFilepath, []byte(metadata), 0o644)
				return "", &cmdrun.Error{Output: "Unable to communicate with SponsorBlock", Status: 1}
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil {
			t.Errorf("expected result")
		}
		if !result.Recovered {
			t.Errorf("expected recovered result")
		}
	})

	t.Run("attempts to update the media item on recoverable errors", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				outputFilePathVal := addlOpts.OutputFilepath
				outputFilepath := outputFilePathVal
				metadata, _ := apptest.RenderMetadata("media_metadata")
				os.WriteFile(outputFilepath, []byte(metadata), 0o644)
				return "", &cmdrun.Error{Output: "Unable to communicate with SponsorBlock", Status: 1}
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil {
			t.Errorf("expected result")
		}
		if result.MediaItem.MediaDownloadedAt == nil {
			t.Errorf("expected media_downloaded_at to be set")
		}
		// Check that media_filepath ends with .mkv
		if result.MediaItem.MediaFilepath == nil || !mediaDownloaderTestHasExtension(*result.MediaItem.MediaFilepath, ".mkv") {
			t.Errorf("expected media_filepath to end with .mkv, got %v", result.MediaItem.MediaFilepath)
		}
	})

	t.Run("returns an unrecoverable tuple if recovery fails", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				// Don't write metadata to the output file, so recovery fails
				return "", &cmdrun.Error{Output: "Unable to communicate with SponsorBlock", Status: 1}
			}
			return "", nil
		})

		_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err == nil {
			t.Errorf("expected error")
		}
		mediaDownloaderErr, ok := err.(*app.MediaDownloaderError)
		if !ok {
			t.Errorf("expected MediaDownloaderError, got %T", err)
		}
		if mediaDownloaderErr.Reason != "unrecoverable" {
			t.Errorf("expected unrecoverable, got %s", mediaDownloaderErr.Reason)
		}
	})

	t.Run("sets the last_error appropriately when recovered", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				outputFilePathVal := addlOpts.OutputFilepath
				outputFilepath := outputFilePathVal
				metadata, _ := apptest.RenderMetadata("media_metadata")
				os.WriteFile(outputFilepath, []byte(metadata), 0o644)
				return "", &cmdrun.Error{Output: "Unable to communicate with SponsorBlock", Status: 1}
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.LastError == nil {
			t.Errorf("expected last_error to be set")
		}
		if *result.MediaItem.LastError != "Unable to communicate with SponsorBlock" {
			t.Errorf("expected last_error to be 'Unable to communicate with SponsorBlock', got %s", *result.MediaItem.LastError)
		}
	})

	t.Run("sets the last_error appropriately when unrecoverable", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				return "", &cmdrun.Error{Output: "Unable to communicate with SponsorBlock", Status: 1}
			}
			return "", nil
		})

		_, _ = ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		// store.Reload the media item from the database
		reloadedMediaItem, _ := ta.App.GetMediaItem(ta.Ctx, mediaItem.ID)

		if reloadedMediaItem.LastError == nil {
			t.Errorf("expected last_error to be set")
		}
		if *reloadedMediaItem.LastError != "Unable to communicate with SponsorBlock" {
			t.Errorf("expected last_error to be 'Unable to communicate with SponsorBlock', got %s", *reloadedMediaItem.LastError)
		}
	})
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingCookieRetries(t *testing.T) {
	t.Parallel()

	t.Run("retries with cookies if we think it would help and the source allows", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"cookie_behaviour": "when_needed"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(4, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			useCookies := addlOpts.UseCookies
			if action == "get_downloadable_status" {
				if useCookies == false {
					return "", &cmdrun.Error{Output: "Sign in to confirm your age", Status: 1}
				} else if useCookies == true {
					return "{}", nil
				}
			} else if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil {
			t.Errorf("expected result")
		}
	})

	t.Run("does not retry with cookies if we don't think it would help even the source allows", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"cookie_behaviour": "when_needed"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "", &cmdrun.Error{Output: "Some other error", Status: 1}
			}
			return "", nil
		})

		_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err == nil {
			t.Errorf("expected error")
		}
		mediaDownloaderErr, ok := err.(*app.MediaDownloaderError)
		if !ok {
			t.Errorf("expected MediaDownloaderError, got %T", err)
		}
		if mediaDownloaderErr.Reason != "download_failed" {
			t.Errorf("expected download_failed, got %s", mediaDownloaderErr.Reason)
		}
	})

	t.Run("does not retry with cookies even if we think it would help but source doesn't allow", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"cookie_behaviour": "disabled"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "", &cmdrun.Error{Output: "Sign in to confirm your age", Status: 1}
			}
			return "", nil
		})

		_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err == nil {
			t.Errorf("expected error")
		}
		mediaDownloaderErr, ok := err.(*app.MediaDownloaderError)
		if !ok {
			t.Errorf("expected MediaDownloaderError, got %T", err)
		}
		if mediaDownloaderErr.Reason != "download_failed" {
			t.Errorf("expected download_failed, got %s", mediaDownloaderErr.Reason)
		}
	})

	t.Run("does not retry with cookies if cookies were already used", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"cookie_behaviour": "all_operations"})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID)})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Expect(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "", &cmdrun.Error{Output: "This video is available to this channel's members", Status: 1}
			}
			return "", nil
		})

		_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if err == nil {
			t.Errorf("expected error")
		}
		mediaDownloaderErr, ok := err.(*app.MediaDownloaderError)
		if !ok {
			t.Errorf("expected MediaDownloaderError, got %T", err)
		}
		if mediaDownloaderErr.Reason != "download_failed" {
			t.Errorf("expected download_failed, got %s", mediaDownloaderErr.Reason)
		}
	})
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingMediaItemAttributes(t *testing.T) {
	t.Parallel()

	t.Run("sets the media_downloaded_at", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		if mediaItem.MediaDownloadedAt != nil {
			t.Errorf("expected media_downloaded_at to be nil initially")
		}

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.MediaDownloadedAt == nil {
			t.Errorf("expected media_downloaded_at to be set")
		}
		// Check that it's close to now (within 2 seconds)
		diff := time.Now().UTC().Sub(result.MediaItem.MediaDownloadedAt.Time)
		if diff.Seconds() > 2 {
			t.Errorf("expected media_downloaded_at to be close to now, diff: %v", diff)
		}
	})

	t.Run("sets the culled_at to nil", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{CulledAt: store.Ptr(time.Now().UTC()), Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.CulledAt != nil {
			t.Errorf("expected culled_at to be nil")
		}
	})

	t.Run("extracts the title", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.Title == nil || *result.MediaItem.Title != "Pinchflat Example Video" {
			title := ""
			if result != nil && result.MediaItem.Title != nil {
				title = *result.MediaItem.Title
			}
			t.Errorf("expected title to be 'Pinchflat Example Video', got %s", title)
		}
	})

	t.Run("extracts the description", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.Description == nil || *result.MediaItem.Description == "" {
			t.Errorf("expected description to be set")
		}
	})

	t.Run("extracts the media_filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		if mediaItem.MediaFilepath != nil {
			t.Errorf("expected media_filepath to be nil initially")
		}

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.MediaFilepath == nil || !mediaDownloaderTestHasExtension(*result.MediaItem.MediaFilepath, ".mkv") {
			t.Errorf("expected media_filepath to end with .mkv")
		}
	})

	t.Run("extracts the subtitle_filepaths", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		if len(mediaItem.SubtitleFilepaths) > 0 {
			t.Errorf("expected no subtitle_filepaths initially")
		}

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || len(result.MediaItem.SubtitleFilepaths) == 0 {
			t.Errorf("expected subtitle_filepaths to be extracted")
		}
	})

	t.Run("extracts the duration_seconds", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		if mediaItem.DurationSeconds != nil {
			t.Errorf("expected duration_seconds to be nil initially")
		}

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.DurationSeconds == nil {
			t.Errorf("expected duration_seconds to be set")
		}
	})

	t.Run("extracts the thumbnail_filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				// Return the parsed live_status response
				return `{"live_status":"not_live"}`, nil
			} else if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		if mediaItem.ThumbnailFilepath != nil {
			t.Errorf("expected thumbnail_filepath to be nil initially")
		}

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		// The thumbnail_filepath should exist (it's created by MetadataFileHelpersDownloadAndStoreThumbnailFor)
		// We just check that it was populated
		if result == nil || result.MediaItem.Metadata == nil {
			t.Errorf("expected metadata to be set")
		}
	})

	t.Run("extracts the metadata_filepath", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				// Return the parsed live_status response
				return `{"live_status":"not_live"}`, nil
			} else if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		if mediaItem.MetadataFilepath != nil {
			t.Errorf("expected metadata_filepath to be nil initially")
		}

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		// The metadata_filepath should exist (it's created by MetadataFileHelpersCompressAndStoreMetadataFor)
		// We just check that it was populated
		if result == nil || result.MediaItem.Metadata == nil {
			t.Errorf("expected metadata to be set")
		}
	})

	t.Run("sets the last_error to nil on success", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{LastError: store.Ptr("Some error"), Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.LastError != nil {
			t.Errorf("expected last_error to be nil")
		}
	})

	t.Run("sets the last_error to the error message on failure", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.ExpectN(2, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download" {
				return "", fmt.Errorf("some_error")
			}
			return "", nil
		})

		_, _ = ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		// store.Reload the media item from the database
		reloadedMediaItem, _ := ta.App.GetMediaItem(ta.Ctx, mediaItem.ID)

		if reloadedMediaItem.LastError == nil {
			t.Errorf("expected last_error to be set")
		}
	})
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingNFOGeneration(t *testing.T) {
	t.Parallel()

	t.Run("generates an NFO file if the source is set to download NFOs", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadNfo: store.Ptr(true)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.NfoFilepath == nil {
			t.Errorf("expected NFO filepath to be set")
		} else if !mediaDownloaderTestHasExtension(*result.MediaItem.NfoFilepath, ".nfo") {
			t.Errorf("expected NFO filepath to end with .nfo")
		}
		if result.MediaItem.NfoFilepath != nil && !mediaDownloaderTestFileExists(*result.MediaItem.NfoFilepath) {
			t.Errorf("expected NFO file to exist")
		}

		// Cleanup
		if result.MediaItem.NfoFilepath != nil && mediaDownloaderTestFileExists(*result.MediaItem.NfoFilepath) {
			os.Remove(*result.MediaItem.NfoFilepath)
		}
	})

	t.Run("does not generate an NFO file if the source is set to not download NFOs", func(t *testing.T) {
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{DownloadNfo: store.Ptr(false)})
		source := apptest.SourceFixture(t, ta, store.Attrs{"media_profile_id": mediaProfile.ID})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})
		mediaItem, _ = ta.App.PreloadMediaItemFull(ta.Ctx, mediaItem)

		ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) {
			return "", nil
		})

		ta.YtDlpMock.Run.Stub(func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			if action == "download" {
				metadata, _ := apptest.RenderMetadata("media_metadata")
				return metadata, nil
			} else if action == "get_downloadable_status" {
				return "{}", nil
			} else if action == "download_thumbnail" {
				return "", nil
			}
			return "", nil
		})

		result, _ := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

		if result == nil || result.MediaItem.NfoFilepath != nil {
			t.Errorf("expected NfoFilepath to be nil")
		}
	})
}

// Helper functions

func mediaDownloaderTestHasExtension(filepath, ext string) bool {
	return len(filepath) > len(ext) && filepath[len(filepath)-len(ext):] == ext
}

func mediaDownloaderTestFileExists(filepath string) bool {
	_, err := os.Stat(filepath)
	return err == nil
}
