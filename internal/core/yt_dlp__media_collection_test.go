package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

func TestYtDlpMediaCollection_GetMediaAttributesForCollection(t *testing.T) {
	t.Run("returns a list of video attributes with no blank elements", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return sourceAttributesReturnFixture() + "\n\n", nil
		})

		result, err := ta.MediaCollectionGetMediaAttributesForCollection(ta.Ctx, channelURL, core.KW{}, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 media items, got %d", len(result))
		}
		if result[0].MediaID != "video1" || result[1].MediaID != "video2" || result[2].MediaID != "video3" {
			t.Errorf("expected media IDs video1, video2, video3, got %v", result)
		}
	})

	t.Run("passes the expected default args", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !opts.Contains(core.Flag("simulate")) ||
				!opts.Contains(core.Flag("skip_download")) ||
				!opts.Contains(core.Flag("ignore_no_formats_error")) ||
				!opts.Contains(core.Flag("no_warnings")) {
				t.Errorf("expected specific default opts, got %v", opts)
			}
			if ot != core.YtDlpMediaIndexingOutputTemplate() {
				t.Errorf("expected output template %q, got %q", core.YtDlpMediaIndexingOutputTemplate(), ot)
			}
			return "", nil
		})

		ta.MediaCollectionGetMediaAttributesForCollection(ta.Ctx, channelURL, core.KW{}, core.KW{})
	})

	t.Run("returns the error straight through when the command fails", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "", &fsutil.CommandError{Output: "Big issue", Status: 1}
		})

		_, err := ta.MediaCollectionGetMediaAttributesForCollection(ta.Ctx, channelURL, core.KW{}, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("passes long additional command options", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !opts.Contains(core.Flag("foo")) {
				t.Errorf("expected opts to contain :foo, got %v", opts)
			}
			return "", nil
		})

		ta.MediaCollectionGetMediaAttributesForCollection(ta.Ctx, channelURL, core.KW{core.Flag("foo")}, core.KW{})
	})

	t.Run("passes additional args to runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			// Check for output_filepath
			if fp, ok := addl.Get("output_filepath"); !ok || fp == "" {
				t.Errorf("expected output_filepath in addl, got %v", addl)
			}
			// Check for use_cookies
			if !addl.Contains(core.Opt("use_cookies", false)) {
				t.Errorf("expected use_cookies: false in addl, got %v", addl)
			}
			return "", nil
		})

		ta.MediaCollectionGetMediaAttributesForCollection(ta.Ctx, channelURL, core.KW{}, core.KW{})
	})

	t.Run("supports an optional file_listener_handler that gets passed a filename", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		handlerCalled := false
		handlerFilename := ""

		handler := func(filename string) {
			handlerCalled = true
			handlerFilename = filename
		}

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "", nil
		})

		ta.MediaCollectionGetMediaAttributesForCollection(ta.Ctx, channelURL, core.KW{}, core.KW{core.Opt("file_listener_handler", handler)})

		if !handlerCalled {
			t.Fatal("expected handler to be called")
		}
		if handlerFilename == "" {
			t.Fatal("expected handler to receive filename")
		}
	})

	t.Run("gracefully handles partially failed responses", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "INVALID\n\n" + sourceAttributesReturnFixture() + "\nINVALID\n", nil
		})

		result, err := ta.MediaCollectionGetMediaAttributesForCollection(ta.Ctx, channelURL, core.KW{}, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("expected 3 media items, got %d", len(result))
		}
	})
}

func TestYtDlpMediaCollection_GetSourceDetails(t *testing.T) {
	t.Run("returns a map with data on success", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return `{
				"channel": "PinchflatTestChannel",
				"channel_id": "UCQH2",
				"playlist_id": "PLQH2",
				"playlist_title": "PinchflatTestChannel - Videos",
				"filename": "path/to/file.mp4"
			}`, nil
		})

		result, err := ta.MediaCollectionGetSourceDetails(ta.Ctx, channelURL, core.KW{}, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result["channel_id"] != "UCQH2" {
			t.Errorf("expected channel_id UCQH2, got %v", result["channel_id"])
		}
		if result["channel_name"] != "PinchflatTestChannel" {
			t.Errorf("expected channel_name PinchflatTestChannel, got %v", result["channel_name"])
		}
		if result["playlist_id"] != "PLQH2" {
			t.Errorf("expected playlist_id PLQH2, got %v", result["playlist_id"])
		}
		if result["playlist_name"] != "PinchflatTestChannel - Videos" {
			t.Errorf("expected playlist_name, got %v", result["playlist_name"])
		}
	})

	t.Run("passes the expected args to the runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if url != channelURL {
				t.Errorf("expected url %q, got %q", channelURL, url)
			}
			if action != "get_source_details" {
				t.Errorf("expected action get_source_details, got %q", action)
			}
			if !opts.Contains(core.Flag("simulate")) ||
				!opts.Contains(core.Flag("skip_download")) ||
				!opts.Contains(core.Flag("ignore_no_formats_error")) ||
				!opts.Contains(core.Opt("playlist_end", 1)) {
				t.Errorf("expected specific opts, got %v", opts)
			}
			if ot != "%(.{channel,channel_id,playlist_id,playlist_title,filename})j" {
				t.Errorf("expected specific output template, got %q", ot)
			}
			return "{}", nil
		})

		ta.MediaCollectionGetSourceDetails(ta.Ctx, channelURL, core.KW{}, core.KW{})
	})

	t.Run("passes custom args to the runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !opts.Contains(core.Opt("foo", core.Flag("bar"))) {
				t.Errorf("expected opts to contain foo: bar, got %v", opts)
			}
			return "{}", nil
		})

		ta.MediaCollectionGetSourceDetails(ta.Ctx, channelURL, core.KW{core.Opt("foo", core.Flag("bar"))}, core.KW{})
	})

	t.Run("passes additional args to the runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !addl.Contains(core.Opt("use_cookies", true)) {
				t.Errorf("expected addl to contain use_cookies: true, got %v", addl)
			}
			return "{}", nil
		})

		ta.MediaCollectionGetSourceDetails(ta.Ctx, channelURL, core.KW{}, core.KW{core.Opt("use_cookies", true)})
	})

	t.Run("returns an error if the runner returns an error", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "", &fsutil.CommandError{Output: "Big issue", Status: 1}
		})

		_, err := ta.MediaCollectionGetSourceDetails(ta.Ctx, channelURL, core.KW{}, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("returns an error if the output is not JSON", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "Not JSON", nil
		})

		_, err := ta.MediaCollectionGetSourceDetails(ta.Ctx, channelURL, core.KW{}, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
		if err.Error() != "Error decoding JSON response" {
			t.Errorf("expected error message 'Error decoding JSON response', got %q", err.Error())
		}
	})
}

func TestYtDlpMediaCollection_GetSourceMetadata(t *testing.T) {
	t.Run("returns a map with data on success", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return `{"channel": "PinchflatTestChannel"}`, nil
		})

		result, err := ta.MediaCollectionGetSourceMetadata(ta.Ctx, channelURL, core.KW{core.Opt("playlist_items", 0)}, core.KW{})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result["channel"] != "PinchflatTestChannel" {
			t.Errorf("expected channel PinchflatTestChannel, got %v", result["channel"])
		}
	})

	t.Run("passes the expected args to the backend runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !opts.Contains(core.Flag("skip_download")) ||
				!opts.Contains(core.Opt("playlist_items", 0)) {
				t.Errorf("expected specific opts, got %v", opts)
			}
			if ot != "playlist:%()j" {
				t.Errorf("expected output template 'playlist:%%()j', got %q", ot)
			}
			return "{}", nil
		})

		ta.MediaCollectionGetSourceMetadata(ta.Ctx, channelURL, core.KW{core.Opt("playlist_items", 0)}, core.KW{})
	})

	t.Run("passes additional args to the runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if !addl.Contains(core.Opt("use_cookies", true)) {
				t.Errorf("expected addl to contain use_cookies: true, got %v", addl)
			}
			return "{}", nil
		})

		ta.MediaCollectionGetSourceMetadata(ta.Ctx, channelURL, core.KW{core.Opt("playlist_items", 0)}, core.KW{core.Opt("use_cookies", true)})
	})

	t.Run("passes custom args to the runner", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			if opts[0].Key != "skip_download" ||
				opts[1].Key != "playlist_items" ||
				opts[2].Key != "real_opt" {
				t.Errorf("expected specific opts, got %v", opts)
			}
			return "{}", nil
		})

		ta.MediaCollectionGetSourceMetadata(ta.Ctx, channelURL, core.KW{core.Opt("playlist_items", 1), core.Opt("real_opt", "yup")}, core.KW{})
	})

	t.Run("blows up if you pass addl opts but don't pass playlist items", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		_, err := ta.MediaCollectionGetSourceMetadata(ta.Ctx, channelURL, core.KW{core.Opt("real_opt", "yup")}, core.KW{})

		if err == nil {
			t.Fatal("expected an error when playlist_items is missing")
		}
	})

	t.Run("returns an error if the runner returns an error", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "", &fsutil.CommandError{Output: "Big issue", Status: 1}
		})

		_, err := ta.MediaCollectionGetSourceMetadata(ta.Ctx, channelURL, core.KW{core.Opt("playlist_items", 0)}, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("returns an error if the output is not JSON", func(t *testing.T) {
		ta := coretest.NewApp(t)
		channelURL := "https://www.youtube.com/c/PinchflatTestChannel"

		ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "Not JSON", nil
		})

		_, err := ta.MediaCollectionGetSourceMetadata(ta.Ctx, channelURL, core.KW{core.Opt("playlist_items", 0)}, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
	})
}

func sourceAttributesReturnFixture() string {
	return `{"id":"video1","title":"Video 1","description":"","original_url":"https://www.youtube.com/watch?v=video1","live_status":"not_live","aspect_ratio":1.0,"duration":60,"upload_date":"20210101","timestamp":1600000000,"playlist_index":1,"filename":"video1.mp4"}
{"id":"video2","title":"Video 2","description":"","original_url":"https://www.youtube.com/watch?v=video2","live_status":"not_live","aspect_ratio":1.0,"duration":60,"upload_date":"20210101","timestamp":1600000000,"playlist_index":2,"filename":"video2.mp4"}
{"id":"video3","title":"Video 3","description":"","original_url":"https://www.youtube.com/watch?v=video3","live_status":"not_live","aspect_ratio":1.0,"duration":60,"upload_date":"20210101","timestamp":1600000000,"playlist_index":3,"filename":"video3.mp4"}`
}
