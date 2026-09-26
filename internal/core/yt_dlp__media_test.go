package core_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
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
			return "", &core.CommandError{Output: "something", Status: 1}
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
			return "", &core.CommandError{Output: "something", Status: 1}
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
			return "", &core.CommandError{Output: "Big issue", Status: 1}
		})

		_, err := ta.YtDlpMediaGetMediaAttributes(ta.Ctx, mediaURL, core.KW{}, core.KW{})

		if err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestYtDlpMedia_IndexingOutputTemplate(t *testing.T) {
	t.Run("contains all the greatest hits", func(t *testing.T) {
		template := core.YtDlpMediaIndexingOutputTemplate()
		expected := "%(.{id,title,live_status,original_url,description,aspect_ratio,duration,upload_date,timestamp,playlist_index,filename})j"

		if template != expected {
			t.Errorf("expected %q, got %q", expected, template)
		}
	})
}

func TestYtDlpMedia_ResponseToStruct(t *testing.T) {
	t.Run("transforms a response into a struct", func(t *testing.T) {
		response := map[string]any{
			"id":             "TiZPUDkDYbk",
			"title":          "Trying to Wheelie Without the Rear Brake",
			"description":    "I'm not sure what I expected.",
			"original_url":   "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"live_status":    "not_live",
			"aspect_ratio":   json.Number("1.0"),
			"duration":       json.Number("60"),
			"upload_date":    "20210101",
			"timestamp":      json.Number("1600000000"),
			"playlist_index": json.Number("1"),
			"filename":       "TiZPUDkDYbk.mp4",
		}

		uploadedAt := time.Date(2020, 9, 13, 12, 26, 40, 0, time.UTC)
		want := &core.YtDlpMedia{
			MediaID:                "TiZPUDkDYbk",
			Title:                  "Trying to Wheelie Without the Rear Brake",
			Description:            "I'm not sure what I expected.",
			OriginalURL:            "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			Livestream:             false,
			ShortFormContent:       core.Ptr(false),
			UploadedAt:             &uploadedAt,
			DurationSeconds:        core.Ptr(60),
			PlaylistIndex:          core.Ptr(1),
			PredictedMediaFilepath: "TiZPUDkDYbk.mp4",
		}
		got := core.YtDlpMediaResponseToStruct(response)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("response_to_struct mismatch\n got: %+v\nwant: %+v", *got, *want)
		}
	})

	t.Run("sets short_form_content to true if the URL contains /shorts/", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/shorts/TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  "20210101",
		}

		result := core.YtDlpMediaResponseToStruct(response)

		if result.ShortFormContent == nil || !*result.ShortFormContent {
			t.Errorf("expected short_form_content to be true")
		}
	})

	t.Run("sets short_form_content to true if the aspect ratio are duration are right", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 0.5,
			"duration":     150.0,
			"upload_date":  "20210101",
		}

		result := core.YtDlpMediaResponseToStruct(response)

		if result.ShortFormContent == nil || !*result.ShortFormContent {
			t.Errorf("expected short_form_content to be true")
		}
	})

	t.Run("sets short_form_content to false otherwise", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  "20210101",
		}

		result := core.YtDlpMediaResponseToStruct(response)

		if result.ShortFormContent == nil || *result.ShortFormContent {
			t.Errorf("expected short_form_content to be false")
		}
	})

	t.Run("doesn't blow up if short form content-related fields are missing", func(t *testing.T) {
		response := map[string]any{
			"original_url": nil,
			"aspect_ratio": nil,
			"duration":     nil,
			"upload_date":  "20210101",
		}

		result := core.YtDlpMediaResponseToStruct(response)

		if result.ShortFormContent != nil {
			t.Errorf("expected short_form_content to be nil, got %v", result.ShortFormContent)
		}
	})

	t.Run("parses the duration", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     60.4,
			"upload_date":  "20210101",
		}

		result := core.YtDlpMediaResponseToStruct(response)

		if result.DurationSeconds == nil || *result.DurationSeconds != 60 {
			t.Errorf("expected duration_seconds to be 60, got %v", result.DurationSeconds)
		}
	})

	t.Run("doesn't blow up if duration is missing", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     nil,
			"upload_date":  "20210101",
		}

		result := core.YtDlpMediaResponseToStruct(response)

		if result.DurationSeconds != nil {
			t.Errorf("expected duration_seconds to be nil, got %v", result.DurationSeconds)
		}
	})

	t.Run("sets livestream to false if the live_status field isn't present", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     60.0,
			"upload_date":  "20210101",
		}

		result := core.YtDlpMediaResponseToStruct(response)

		if result.Livestream {
			t.Errorf("expected livestream to be false, got %v", result.Livestream)
		}
	})

	t.Run("doesn't blow up if playlist_index is missing", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     nil,
			"upload_date":  "20210101",
		}

		result := core.YtDlpMediaResponseToStruct(response)

		if result.PlaylistIndex == nil || *result.PlaylistIndex != 0 {
			t.Errorf("expected playlist_index to be 0, got %v", result.PlaylistIndex)
		}
	})
}

func TestYtDlpMedia_ResponseToStructUploadedAt(t *testing.T) {
	t.Run("parses the upload date from the timestamp if present", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  "20210101",
			"timestamp":    json.Number("1600000000"),
		}

		result := core.YtDlpMediaResponseToStruct(response)

		expectedDate := time.Unix(1600000000, 0).UTC()
		if result.UploadedAt == nil || !result.UploadedAt.Equal(expectedDate) {
			t.Errorf("expected upload date %v, got %v", expectedDate, result.UploadedAt)
		}
	})

	t.Run("parses the upload date from the uploaded_at if timestamp is present but nil", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  "20210101",
			"timestamp":    nil,
		}

		result := core.YtDlpMediaResponseToStruct(response)

		expectedDate, _ := time.Parse("20060102", "20210101")
		if result.UploadedAt == nil || result.UploadedAt.Format("2006-01-02") != expectedDate.Format("2006-01-02") {
			t.Errorf("expected upload date around 2021-01-01, got %v", result.UploadedAt)
		}
	})

	t.Run("parses the upload date from the uploaded_at if timestamp absent", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  "20210101",
		}

		result := core.YtDlpMediaResponseToStruct(response)

		expectedDate, _ := time.Parse("20060102", "20210101")
		if result.UploadedAt == nil || result.UploadedAt.Format("2006-01-02") != expectedDate.Format("2006-01-02") {
			t.Errorf("expected upload date around 2021-01-01, got %v", result.UploadedAt)
		}
	})

	t.Run("doesn't blow up if upload date is missing", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  nil,
		}

		result := core.YtDlpMediaResponseToStruct(response)

		if result.UploadedAt != nil {
			t.Errorf("expected upload date to be nil, got %v", result.UploadedAt)
		}
	})
}

func renderMetadata(t *testing.T, name string) string {
	data, err := os.ReadFile(coretest.RepoPath(filepath.Join("test/support/files", name+".json")))
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
