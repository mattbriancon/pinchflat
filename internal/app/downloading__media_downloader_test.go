package app_test

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/cmdrun"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// ytCall is one yt-dlp runner invocation.
type ytCall struct {
	url, outputTemplate string
	opts                ytdlp.Args
	addl                ytdlp.CallOptions
}

// dlActions maps yt-dlp actions to handlers.
type dlActions map[string]func(ytCall) (string, error)

func ret(s string) func(ytCall) (string, error) {
	return func(ytCall) (string, error) { return s, nil }
}

func retErr(err error) func(ytCall) (string, error) {
	return func(ytCall) (string, error) { return "", err }
}

func retMetadata(ytCall) (string, error) { return apptest.RenderMetadata("media_metadata") }

// writesOutputThenFails writes the metadata to the output file, like yt-dlp
// does before a recoverable failure, then fails with a SponsorBlock error.
func writesOutputThenFails(c ytCall) (string, error) {
	metadata, err := apptest.RenderMetadata("media_metadata")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(c.addl.OutputFilepath, []byte(metadata), 0o644); err != nil {
		return "", err
	}
	return "", &cmdrun.Error{Output: "Unable to communicate with SponsorBlock", Status: 1}
}

// downloadMock returns a yt-dlp mock for a successful download, with
// override replacing the handlers of individual actions. Unhandled actions
// (e.g. download_thumbnail) return "".
func downloadMock(override dlActions) func(url, action string, opts ytdlp.Args, outputTemplate string, addl ytdlp.CallOptions) (string, error) {
	m := dlActions{"get_downloadable_status": ret("{}"), "download": retMetadata}
	for k, v := range override {
		m[k] = v
	}
	return func(url, action string, opts ytdlp.Args, outputTemplate string, addl ytdlp.CallOptions) (string, error) {
		if f, ok := m[action]; ok {
			return f(ytCall{url, outputTemplate, opts, addl})
		}
		return "", nil
	}
}

// dlSetup creates the fixtures for a download: a profile, source and
// preloaded media item, with the HTTP mock stubbed out.
func dlSetup(t *testing.T, profile store.MediaProfileParams, src store.SourceParams, item store.MediaItemParams) (*apptest.TestApp, *store.MediaItem) {
	t.Helper()
	ta := apptest.NewApp(t)
	src.MediaProfileID = store.Ptr(apptest.MediaProfileFixture(t, ta, profile).ID)
	item.SourceID = store.Ptr(apptest.SourceFixture(t, ta, src).ID)
	mediaItem, err := ta.App.PreloadMediaItemFull(ta.Ctx, apptest.MediaItemFixture(t, ta, item))
	must(t, err)
	ta.HTTPMock.Get.Stub(func(url string, headers http.Header) (string, error) { return "", nil })
	return ta, mediaItem
}

// dlOK downloads the media item and expects a result.
func dlOK(t *testing.T, ta *apptest.TestApp, mediaItem *store.MediaItem, o app.DownloadOverrides) *app.MediaDownloaderResult {
	t.Helper()
	result, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, o)
	must(t, err)
	if result == nil {
		t.Fatal("expected result")
	}
	return result
}

func wantDownloaderError(t *testing.T, err error, reason string) {
	t.Helper()
	dErr, ok := err.(*app.MediaDownloaderError)
	if !ok {
		t.Fatalf("expected MediaDownloaderError, got %T (%v)", err, err)
	}
	if dErr.Reason != reason {
		t.Errorf("expected %s, got %s", reason, dErr.Reason)
	}
}

func hasSuffix(s *string, suffix string) bool { return s != nil && strings.HasSuffix(*s, suffix) }

func TestMediaDownloader_DownloadForMediaItem(t *testing.T) {
	t.Parallel()

	t.Run("calls the backend runner", func(t *testing.T) {
		ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{}, store.MediaItemParams{Clear: store.ClearMediaFilepath})

		ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, outputTemplate string, addlOpts ytdlp.CallOptions) (string, error) {
			switch action {
			case "get_downloadable_status":
				return "{}", nil
			case "download_thumbnail":
				return "", nil
			case "download":
				if url != mediaItem.OriginalURL {
					t.Errorf("expected URL %s, got %s", mediaItem.OriginalURL, url)
				}
				if outputTemplate != "after_move:%()j" {
					t.Errorf("expected outputTemplate 'after_move:%%()j', got %s", outputTemplate)
				}
				if addlOpts.OutputFilepath == "" {
					t.Errorf("expected output_filepath in addlOpts")
				}
				return apptest.RenderMetadata("media_metadata")
			}
			return "", fmt.Errorf("unexpected action: %s", action)
		})

		dlOK(t, ta, mediaItem, app.DownloadOverrides{})
	})

	t.Run("doesn't share the download's output file with the status check", func(t *testing.T) {
		ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{}, store.MediaItemParams{Clear: store.ClearMediaFilepath})

		var statusOutput, downloadOutput string
		ta.YtDlpMock.Run.ExpectN(3, downloadMock(dlActions{
			"get_downloadable_status": func(c ytCall) (string, error) {
				statusOutput = c.addl.OutputFilepath
				return "{}", nil
			},
			"download": func(c ytCall) (string, error) {
				downloadOutput = c.addl.OutputFilepath
				return retMetadata(c)
			},
		}))

		dlOK(t, ta, mediaItem, app.DownloadOverrides{})
		if statusOutput != "" && statusOutput == downloadOutput {
			t.Errorf("status check and download both wrote to %s", statusOutput)
		}
	})

	t.Run("saves the metadata filepath to the database", func(t *testing.T) {
		ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{}, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		ta.YtDlpMock.Run.ExpectN(3, downloadMock(nil))

		if mediaItem.Metadata != nil {
			t.Errorf("expected no metadata initially")
		}

		result := dlOK(t, ta, mediaItem, app.DownloadOverrides{})

		if result.MediaItem.Metadata == nil {
			t.Fatalf("expected metadata to be saved")
		}
		if result.MediaItem.Metadata.MetadataFilepath == "" {
			t.Errorf("expected metadata_filepath")
		}
		if result.MediaItem.Metadata.ThumbnailFilepath == "" {
			t.Errorf("expected thumbnail_filepath")
		}
	})

	t.Run("calls the download runner if the media is currently downloadable", func(t *testing.T) {
		ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{}, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		ta.YtDlpMock.Run.ExpectN(3, downloadMock(dlActions{"get_downloadable_status": ret(`{"live_status": "was_live"}`)}))

		dlOK(t, ta, mediaItem, app.DownloadOverrides{})
	})

	t.Run("includes override opts if specified", func(t *testing.T) {
		ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{}, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		ta.YtDlpMock.Run.ExpectN(3, downloadMock(dlActions{"download": func(c ytCall) (string, error) {
			wantOpts(t, c.opts, []string{"no_force_overwrites"}, []string{"force_overwrites"})
			return retMetadata(c)
		}}))

		dlOK(t, ta, mediaItem, app.DownloadOverrides{OverwriteBehaviour: "no_force_overwrites"})
	})

	t.Run("sets use_cookies according to the source's cookie behaviour", func(t *testing.T) {
		for _, c := range []struct {
			name      string
			behaviour store.SourceCookieBehaviour
			want      bool
		}{
			{"sets use_cookies if the source uses cookies", store.SourceCookieBehaviourAllOperations, true},
			{"does not set use_cookies if the source uses cookies when needed", store.SourceCookieBehaviourWhenNeeded, false},
			{"does not set use_cookies if the source does not use cookies", store.SourceCookieBehaviourDisabled, false},
		} {
			t.Run(c.name, func(t *testing.T) {
				ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{CookieBehaviour: store.Ptr(c.behaviour)}, store.MediaItemParams{Clear: store.ClearMediaFilepath})
				inner := downloadMock(nil)
				ta.YtDlpMock.Run.ExpectN(3, func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
					if addl.UseCookies != c.want {
						t.Errorf("expected use_cookies to be %v, got %v", c.want, addl.UseCookies)
					}
					return inner(url, action, opts, ot, addl)
				})

				dlOK(t, ta, mediaItem, app.DownloadOverrides{})
			})
		}
	})

	t.Run("retries with cookies if we think it would help and the source allows", func(t *testing.T) {
		ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{CookieBehaviour: store.Ptr(store.SourceCookieBehaviourWhenNeeded)}, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		ta.YtDlpMock.Run.ExpectN(4, downloadMock(dlActions{"get_downloadable_status": func(c ytCall) (string, error) {
			if !c.addl.UseCookies {
				return "", &cmdrun.Error{Output: "Sign in to confirm your age", Status: 1}
			}
			return "{}", nil
		}}))

		dlOK(t, ta, mediaItem, app.DownloadOverrides{})
	})
}

// Each case runs a download whose yt-dlp calls fail and expects a
// MediaDownloaderError with the given reason.
func TestMediaDownloader_DownloadForMediaItem_Errors(t *testing.T) {
	t.Parallel()

	cmdErr := func(msg string) error { return &cmdrun.Error{Output: msg, Status: 1} }
	failsOn := func(action string, err error) dlActions {
		return dlActions{action: retErr(err), "get_downloadable_status": ret("{}")}
	}
	statusFails := func(err error) dlActions { return dlActions{"get_downloadable_status": retErr(err)} }
	cookies := func(b store.SourceCookieBehaviour) store.SourceParams {
		return store.SourceParams{CookieBehaviour: store.Ptr(b)}
	}

	for _, c := range []struct {
		name    string
		src     store.SourceParams
		actions dlActions
		calls   int
		reason  string
	}{
		{"errors for non-downloadable media are passed through (does not call the download runner)", store.SourceParams{}, dlActions{"get_downloadable_status": ret(`{"live_status": "is_live"}`)}, 1, "unsuitable_for_download"},
		{"non-recoverable errors are passed through", store.SourceParams{}, failsOn("download", cmdErr("some_error")), 2, "download_failed"},
		{"unknown errors are passed through", store.SourceParams{}, failsOn("download", fmt.Errorf("some_error")), 2, "unknown"},
		{"returns unexpected errors from the download status determination method", store.SourceParams{}, statusFails(fmt.Errorf("what_tha")), 1, "unknown"},
		{"returns an unrecoverable tuple if recovery fails", store.SourceParams{}, failsOn("download", cmdErr("Unable to communicate with SponsorBlock")), 2, "unrecoverable"},
		{"does not retry with cookies if we don't think it would help even the source allows", cookies(store.SourceCookieBehaviourWhenNeeded), statusFails(cmdErr("Some other error")), 1, "download_failed"},
		{"does not retry with cookies even if we think it would help but source doesn't allow", cookies(store.SourceCookieBehaviourDisabled), statusFails(cmdErr("Sign in to confirm your age")), 1, "download_failed"},
		{"does not retry with cookies if cookies were already used", cookies(store.SourceCookieBehaviourAllOperations), statusFails(cmdErr("This video is available to this channel's members")), 1, "download_failed"},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, c.src, store.MediaItemParams{})
			ta.YtDlpMock.Run.ExpectN(c.calls, downloadMock(c.actions))

			_, err := ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

			wantDownloaderError(t, err, c.reason)
		})
	}
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingNonCookieRetries(t *testing.T) {
	t.Parallel()

	t.Run("recovers from recoverable errors", func(t *testing.T) {
		ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{}, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		ta.YtDlpMock.Run.ExpectN(3, downloadMock(dlActions{"download": writesOutputThenFails}))

		result := dlOK(t, ta, mediaItem, app.DownloadOverrides{})

		t.Run("returns a recovered tuple on recoverable errors", func(t *testing.T) {
			if !result.Recovered {
				t.Errorf("expected recovered result")
			}
		})
		t.Run("attempts to update the media item on recoverable errors", func(t *testing.T) {
			if result.MediaItem.MediaDownloadedAt == nil {
				t.Errorf("expected media_downloaded_at to be set")
			}
			if !hasSuffix(result.MediaItem.MediaFilepath, ".mkv") {
				t.Errorf("expected media_filepath to end with .mkv, got %v", result.MediaItem.MediaFilepath)
			}
		})
		t.Run("sets the last_error appropriately when recovered", func(t *testing.T) {
			if result.MediaItem.LastError == nil || *result.MediaItem.LastError != "Unable to communicate with SponsorBlock" {
				t.Errorf("expected last_error to be 'Unable to communicate with SponsorBlock', got %v", result.MediaItem.LastError)
			}
		})
	})

	for _, c := range []struct {
		name string
		err  error
		want string // expected last_error; "" only requires it to be set
	}{
		{"sets the last_error appropriately when unrecoverable", &cmdrun.Error{Output: "Unable to communicate with SponsorBlock", Status: 1}, "Unable to communicate with SponsorBlock"},
		{"sets the last_error to the error message on failure", fmt.Errorf("some_error"), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{}, store.MediaItemParams{})
			ta.YtDlpMock.Run.ExpectN(2, downloadMock(dlActions{"download": retErr(c.err)}))

			_, _ = ta.App.MediaDownloaderDownloadForMediaItem(ta.Ctx, mediaItem, app.DownloadOverrides{})

			reloaded, err := ta.App.GetMediaItem(ta.Ctx, mediaItem.ID)
			must(t, err)
			if reloaded.LastError == nil {
				t.Fatalf("expected last_error to be set")
			}
			if c.want != "" && *reloaded.LastError != c.want {
				t.Errorf("expected last_error to be %q, got %s", c.want, *reloaded.LastError)
			}
		})
	}
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingMediaItemAttributes(t *testing.T) {
	t.Parallel()

	cleared := store.MediaItemParams{Clear: store.ClearMediaFilepath}
	for _, c := range []struct {
		name    string
		item    store.MediaItemParams
		status  string                      // downloadable-status response, "{}" if empty
		initial func(*store.MediaItem) bool // must hold before the download, if set
		ok      func(*store.MediaItem) bool // must hold on the downloaded item
	}{
		{"sets the media_downloaded_at", cleared, "", func(m *store.MediaItem) bool { return m.MediaDownloadedAt == nil },
			func(m *store.MediaItem) bool {
				return m.MediaDownloadedAt != nil && time.Now().UTC().Sub(m.MediaDownloadedAt.Time).Seconds() <= 2
			}},
		{"sets the culled_at to nil", store.MediaItemParams{CulledAt: store.Ptr(time.Now().UTC()), Clear: store.ClearMediaFilepath}, "", nil,
			func(m *store.MediaItem) bool { return m.CulledAt == nil }},
		{"extracts the title", cleared, "", nil,
			func(m *store.MediaItem) bool { return m.Title != nil && *m.Title == "Pinchflat Example Video" }},
		{"extracts the description", cleared, "", nil,
			func(m *store.MediaItem) bool { return m.Description != nil && *m.Description != "" }},
		{"extracts the media_filepath", cleared, "", func(m *store.MediaItem) bool { return m.MediaFilepath == nil },
			func(m *store.MediaItem) bool { return hasSuffix(m.MediaFilepath, ".mkv") }},
		{"extracts the subtitle_filepaths", cleared, "", func(m *store.MediaItem) bool { return len(m.SubtitleFilepaths) == 0 },
			func(m *store.MediaItem) bool { return len(m.SubtitleFilepaths) > 0 }},
		{"extracts the duration_seconds", cleared, "", func(m *store.MediaItem) bool { return m.DurationSeconds == nil },
			func(m *store.MediaItem) bool { return m.DurationSeconds != nil }},
		{"extracts the thumbnail_filepath", cleared, `{"live_status":"not_live"}`, func(m *store.MediaItem) bool { return m.ThumbnailFilepath == nil },
			func(m *store.MediaItem) bool { return m.Metadata != nil }},
		{"extracts the metadata_filepath", cleared, `{"live_status":"not_live"}`, func(m *store.MediaItem) bool { return m.MetadataFilepath == nil },
			func(m *store.MediaItem) bool { return m.Metadata != nil }},
		{"sets the last_error to nil on success", store.MediaItemParams{LastError: store.Ptr("Some error"), Clear: store.ClearMediaFilepath}, "", nil,
			func(m *store.MediaItem) bool { return m.LastError == nil }},
	} {
		t.Run(c.name, func(t *testing.T) {
			ta, mediaItem := dlSetup(t, store.MediaProfileParams{}, store.SourceParams{}, c.item)
			status := c.status
			if status == "" {
				status = "{}"
			}
			ta.YtDlpMock.Run.ExpectN(3, downloadMock(dlActions{"get_downloadable_status": ret(status)}))
			if c.initial != nil && !c.initial(mediaItem) {
				t.Errorf("%s: initial state not as expected", c.name)
			}

			result := dlOK(t, ta, mediaItem, app.DownloadOverrides{})

			if !c.ok(result.MediaItem) {
				t.Errorf("%s: unexpected media item %+v", c.name, result.MediaItem)
			}
		})
	}
}

func TestMediaDownloader_DownloadForMediaItem_WhenTestingNFOGeneration(t *testing.T) {
	t.Parallel()

	t.Run("generates an NFO file if the source is set to download NFOs", func(t *testing.T) {
		ta, mediaItem := dlSetup(t, store.MediaProfileParams{DownloadNfo: store.Ptr(true)}, store.SourceParams{}, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		ta.YtDlpMock.Run.Stub(downloadMock(nil))

		result := dlOK(t, ta, mediaItem, app.DownloadOverrides{})

		nfo := result.MediaItem.NfoFilepath
		if nfo == nil {
			t.Fatalf("expected NFO filepath to be set")
		}
		if !hasSuffix(nfo, ".nfo") {
			t.Errorf("expected NFO filepath to end with .nfo")
		}
		if _, err := os.Stat(*nfo); err != nil {
			t.Errorf("expected NFO file to exist: %v", err)
		} else {
			os.Remove(*nfo)
		}
	})

	t.Run("does not generate an NFO file if the source is set to not download NFOs", func(t *testing.T) {
		ta, mediaItem := dlSetup(t, store.MediaProfileParams{DownloadNfo: store.Ptr(false)}, store.SourceParams{}, store.MediaItemParams{Clear: store.ClearMediaFilepath})
		ta.YtDlpMock.Run.Stub(downloadMock(nil))

		if result := dlOK(t, ta, mediaItem, app.DownloadOverrides{}); result.MediaItem.NfoFilepath != nil {
			t.Errorf("expected NfoFilepath to be nil")
		}
	})
}
