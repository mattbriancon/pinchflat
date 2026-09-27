package core_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

func TestYtDlpMedia_Download(t *testing.T) {
	t.Run("calls the backend runner with the expected arguments", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if action != "download" {
				t.Errorf("expected action 'download', got %q", action)
			}
			if len(opts) != 1 || !opts.Contains(core.Flag("no_simulate")) {
				t.Errorf("expected opts to be [:no_simulate], got %v", opts)
			}
			if ot != "after_move:%()j" {
				t.Errorf("expected output template 'after_move:%%()j', got %q", ot)
			}
			if len(addl) != 0 {
				t.Errorf("expected empty addl, got %v", addl)
			}

			return renderMetadata(t, "media_metadata"), nil
		})

		result, err := ta.YtDlpMediaDownload(ta.Ctx, mediaURL, core.KW{}, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
	})

	t.Run("passes along custom command args", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			hasNoSimulate := opts.Contains(core.Flag("no_simulate"))
			hasCustom := opts.Contains(core.Flag("custom_arg"))
			if !hasNoSimulate || !hasCustom {
				t.Errorf("expected opts to contain [:no_simulate, :custom_arg], got %v", opts)
			}
			return "{}", nil
		})

		ta.YtDlpMediaDownload(ta.Ctx, mediaURL, core.KW{core.Flag("custom_arg")}, core.KW{})
	})

	t.Run("passes along additional options", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !addl.Contains(core.Opt("addl_arg", true)) {
				t.Errorf("expected addl to contain addl_arg: true, got %v", addl)
			}
			return "{}", nil
		})

		ta.YtDlpMediaDownload(ta.Ctx, mediaURL, core.KW{}, core.KW{core.Opt("addl_arg", true)})
	})

	t.Run("parses and returns the generated file as JSON", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return renderMetadata(t, "media_metadata"), nil
		})

		result, err := ta.YtDlpMediaDownload(ta.Ctx, mediaURL, core.KW{}, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if title, ok := result["title"].(string); !ok || title != "Pinchflat Example Video" {
			t.Errorf("expected title 'Pinchflat Example Video', got %v", result["title"])
		}
	})

	t.Run("returns errors", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "", &fsutil.CommandError{Output: "something", Status: 1}
		})

		_, err := ta.YtDlpMediaDownload(ta.Ctx, mediaURL, core.KW{}, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestYtDlpMedia_GetDownloadableStatus(t *testing.T) {
	t.Run("returns :downloadable if the media was never live", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return `{"live_status":"not_live"}`, nil
		})

		result, err := ta.YtDlpMediaGetDownloadableStatus(ta.Ctx, mediaURL, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "downloadable" {
			t.Errorf("expected 'downloadable', got %q", result)
		}
	})

	t.Run("returns :downloadable if the media was live and has been processed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return `{"live_status":"was_live"}`, nil
		})

		result, err := ta.YtDlpMediaGetDownloadableStatus(ta.Ctx, mediaURL, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "downloadable" {
			t.Errorf("expected 'downloadable', got %q", result)
		}
	})

	t.Run("returns :downloadable if the media's live_status is nil", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return `{"live_status":null}`, nil
		})

		result, err := ta.YtDlpMediaGetDownloadableStatus(ta.Ctx, mediaURL, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "downloadable" {
			t.Errorf("expected 'downloadable', got %q", result)
		}
	})

	t.Run("returns :ignorable if the media is currently live", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return `{"live_status":"is_live"}`, nil
		})

		result, err := ta.YtDlpMediaGetDownloadableStatus(ta.Ctx, mediaURL, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "ignorable" {
			t.Errorf("expected 'ignorable', got %q", result)
		}
	})

	t.Run("returns :ignorable if the media is scheduled to be live", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return `{"live_status":"is_upcoming"}`, nil
		})

		result, err := ta.YtDlpMediaGetDownloadableStatus(ta.Ctx, mediaURL, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "ignorable" {
			t.Errorf("expected 'ignorable', got %q", result)
		}
	})

	t.Run("returns :ignorable if the media was live but hasn't been processed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return `{"live_status":"post_live"}`, nil
		})

		result, err := ta.YtDlpMediaGetDownloadableStatus(ta.Ctx, mediaURL, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "ignorable" {
			t.Errorf("expected 'ignorable', got %q", result)
		}
	})

	t.Run("returns an error if the downloadable status can't be determined", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return `{"live_status":"what_tha"}`, nil
		})

		result, err := ta.YtDlpMediaGetDownloadableStatus(ta.Ctx, mediaURL, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
		if result != "" {
			t.Errorf("expected empty string on error, got %q", result)
		}
	})

	t.Run("optionally accepts additional args", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !addl.Contains(core.Opt("addl_arg", true)) {
				t.Errorf("expected addl to contain addl_arg: true, got %v", addl)
			}
			return `{"live_status":"not_live"}`, nil
		})

		ta.YtDlpMediaGetDownloadableStatus(ta.Ctx, mediaURL, core.KW{core.Opt("addl_arg", true)})
	})
}

func TestYtDlpMedia_DownloadThumbnail(t *testing.T) {
	t.Run("calls the backend runner with the expected arguments", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			expectedOpts := core.KW{
				core.Flag("no_simulate"),
				core.Flag("skip_download"),
				core.Flag("write_thumbnail"),
				core.Opt("convert_thumbnail", "jpg"),
			}
			if !optsEqual(opts, expectedOpts) {
				t.Errorf("expected opts %v, got %v", expectedOpts, opts)
			}
			if ot != "after_move:%()j" {
				t.Errorf("expected output template 'after_move:%%()j', got %q", ot)
			}
			return "", nil
		})

		ta.YtDlpMediaDownloadThumbnail(ta.Ctx, mediaURL, core.KW{}, core.KW{})
	})

	t.Run("passes along custom command args", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			hasCustom := opts.Contains(core.Flag("custom_arg"))
			if !hasCustom {
				t.Errorf("expected opts to contain :custom_arg, got %v", opts)
			}
			return "{}", nil
		})

		ta.YtDlpMediaDownloadThumbnail(ta.Ctx, mediaURL, core.KW{core.Flag("custom_arg")}, core.KW{})
	})

	t.Run("passes along additional options", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !addl.Contains(core.Opt("addl_arg", true)) {
				t.Errorf("expected addl to contain addl_arg: true, got %v", addl)
			}
			return "{}", nil
		})

		ta.YtDlpMediaDownloadThumbnail(ta.Ctx, mediaURL, core.KW{}, core.KW{core.Opt("addl_arg", true)})
	})

	t.Run("returns errors", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "", &fsutil.CommandError{Output: "something", Status: 1}
		})

		_, err := ta.YtDlpMediaDownloadThumbnail(ta.Ctx, mediaURL, core.KW{}, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestYtDlpMedia_GetMediaAttributes(t *testing.T) {
	t.Run("returns a list of video attributes", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return mediaAttributesReturnFixture(), nil
		})

		result, err := ta.YtDlpMediaGetMediaAttributes(ta.Ctx, mediaURL, core.KW{}, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Description == "" && result.MediaID == "" && result.OriginalURL == "" && result.Title == "" {
			t.Error("expected result to have some fields set")
		}
	})

	t.Run("it passes the expected default args", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !opts.Contains(core.Flag("simulate")) || !opts.Contains(core.Flag("skip_download")) {
				t.Errorf("expected opts to contain [:simulate, :skip_download], got %v", opts)
			}
			if ot != core.YtDlpMediaIndexingOutputTemplate() {
				t.Errorf("expected output template %q, got %q", core.YtDlpMediaIndexingOutputTemplate(), ot)
			}
			return mediaAttributesReturnFixture(), nil
		})

		ta.YtDlpMediaGetMediaAttributes(ta.Ctx, mediaURL, core.KW{}, core.KW{})
	})

	t.Run("passes along additional command options", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !opts.Contains(core.Flag("simulate")) || !opts.Contains(core.Flag("skip_download")) || !opts.Contains(core.Flag("custom_arg")) {
				t.Errorf("expected opts to contain [:simulate, :skip_download, :custom_arg], got %v", opts)
			}
			return mediaAttributesReturnFixture(), nil
		})

		ta.YtDlpMediaGetMediaAttributes(ta.Ctx, mediaURL, core.KW{core.Flag("custom_arg")}, core.KW{})
	})

	t.Run("passes along additional options", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !addl.Contains(core.Opt("addl_arg", true)) {
				t.Errorf("expected addl to contain addl_arg: true, got %v", addl)
			}
			return mediaAttributesReturnFixture(), nil
		})

		ta.YtDlpMediaGetMediaAttributes(ta.Ctx, mediaURL, core.KW{}, core.KW{core.Opt("addl_arg", true)})
	})

	t.Run("returns the error straight through when the command fails", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaURL := "https://www.youtube.com/watch?v=TiZPUDkDYbk"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "", &fsutil.CommandError{Output: "Big issue", Status: 1}
		})

		_, err := ta.YtDlpMediaGetMediaAttributes(ta.Ctx, mediaURL, core.KW{}, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
	})
}

func renderMetadata(t *testing.T, name string) string {
	data, err := os.ReadFile(coretest.RepoPath(filepath.Join("testdata/support/files", name+".json")))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func mediaAttributesReturnFixture() string {
	return `{
		"id": "TiZPUDkDYbk",
		"title": "Trying to Wheelie Without the Rear Brake",
		"description": "I'm not sure what I expected.",
		"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
		"live_status": "not_live",
		"aspect_ratio": 1.0,
		"duration": 60,
		"upload_date": "20210101",
		"timestamp": 1600000000,
		"playlist_index": 1,
		"filename": "TiZPUDkDYbk.mp4"
	}`
}

func optsEqual(a, b core.KW) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a.Contains(b[i]) {
			return false
		}
	}
	return true
}
