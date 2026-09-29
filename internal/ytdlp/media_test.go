package ytdlp_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

func ptr[T any](v T) *T { return &v }

func TestIndexingOutputTemplate(t *testing.T) {
	template := ytdlp.IndexingOutputTemplate()
	expected := "%(.{id,title,live_status,original_url,description,aspect_ratio,duration,upload_date,timestamp,playlist_index,filename})j"
	if template != expected {
		t.Errorf("expected %q, got %q", expected, template)
	}
}

func TestResponseToStruct(t *testing.T) {
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
		want := &ytdlp.Media{
			MediaID:                "TiZPUDkDYbk",
			Title:                  "Trying to Wheelie Without the Rear Brake",
			Description:            "I'm not sure what I expected.",
			OriginalURL:            "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			Livestream:             false,
			ShortFormContent:       ptr(false),
			UploadedAt:             &uploadedAt,
			DurationSeconds:        ptr(60),
			PlaylistIndex:          ptr(1),
			PredictedMediaFilepath: "TiZPUDkDYbk.mp4",
		}
		got := ytdlp.ResponseToStruct(response)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("ResponseToStruct mismatch\n got: %+v\nwant: %+v", *got, *want)
		}
	})

	t.Run("sets short_form_content to true if the URL contains /shorts/", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/shorts/TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  "20210101",
		}
		result := ytdlp.ResponseToStruct(response)
		if result.ShortFormContent == nil || !*result.ShortFormContent {
			t.Errorf("expected short_form_content to be true")
		}
	})

	t.Run("sets short_form_content to true if the aspect ratio and duration are right", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 0.5,
			"duration":     150.0,
			"upload_date":  "20210101",
		}
		result := ytdlp.ResponseToStruct(response)
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
		result := ytdlp.ResponseToStruct(response)
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
		result := ytdlp.ResponseToStruct(response)
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
		result := ytdlp.ResponseToStruct(response)
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
		result := ytdlp.ResponseToStruct(response)
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
		result := ytdlp.ResponseToStruct(response)
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
		result := ytdlp.ResponseToStruct(response)
		if result.PlaylistIndex == nil || *result.PlaylistIndex != 0 {
			t.Errorf("expected playlist_index to be 0, got %v", result.PlaylistIndex)
		}
	})
}

func TestResponseToStructUploadedAt(t *testing.T) {
	t.Run("parses the upload date from the timestamp if present", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  "20210101",
			"timestamp":    json.Number("1600000000"),
		}
		result := ytdlp.ResponseToStruct(response)
		expectedDate := time.Unix(1600000000, 0).UTC()
		if result.UploadedAt == nil || !result.UploadedAt.Equal(expectedDate) {
			t.Errorf("expected upload date %v, got %v", expectedDate, result.UploadedAt)
		}
	})

	t.Run("parses the upload date from upload_date if timestamp is present but nil", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  "20210101",
			"timestamp":    nil,
		}
		result := ytdlp.ResponseToStruct(response)
		expectedDate, _ := time.Parse("20060102", "20210101")
		if result.UploadedAt == nil || result.UploadedAt.Format("2006-01-02") != expectedDate.Format("2006-01-02") {
			t.Errorf("expected upload date around 2021-01-01, got %v", result.UploadedAt)
		}
	})

	t.Run("parses the upload date from upload_date if timestamp absent", func(t *testing.T) {
		response := map[string]any{
			"original_url": "https://www.youtube.com/watch?v=TiZPUDkDYbk",
			"aspect_ratio": 1.0,
			"duration":     61.0,
			"upload_date":  "20210101",
		}
		result := ytdlp.ResponseToStruct(response)
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
		result := ytdlp.ResponseToStruct(response)
		if result.UploadedAt != nil {
			t.Errorf("expected upload date to be nil, got %v", result.UploadedAt)
		}
	})
}
