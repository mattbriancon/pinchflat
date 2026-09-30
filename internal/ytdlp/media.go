package ytdlp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Media is a piece of media parsed from yt-dlp's JSON output. It isn't a
// database table, just a transfer struct between yt-dlp and the caller.
type Media struct {
	MediaID                string     `json:"media_id"`
	Title                  string     `json:"title"`
	Description            string     `json:"description"`
	OriginalURL            string     `json:"original_url"`
	Livestream             bool       `json:"livestream"`
	ShortFormContent       *bool      `json:"short_form_content"`
	UploadedAt             *time.Time `json:"uploaded_at"`
	DurationSeconds        *int       `json:"duration_seconds"`
	PredictedMediaFilepath string     `json:"predicted_media_filepath"`
	PlaylistIndex          *int       `json:"playlist_index"`
	// RequiresMembership is set for channel-members-only and YouTube
	// Premium-only media, which can't be downloaded without a paid account.
	RequiresMembership bool `json:"requires_membership"`
}

// IndexingOutputTemplate is the --print-to-file template used to fetch a
// Media's attributes.
func IndexingOutputTemplate() string {
	return "%(.{id,title,live_status,original_url,description,aspect_ratio,duration,upload_date,timestamp,playlist_index,filename,availability})j"
}

// ResponseToStruct converts a decoded yt-dlp JSON response into a Media.
func ResponseToStruct(response map[string]any) *Media {
	media := &Media{
		MediaID:     getString(response, "id"),
		Title:       getString(response, "title"),
		Description: getString(response, "description"),
		OriginalURL: getString(response, "original_url"),
		Livestream:  isLivestream(response),
	}

	if duration := getFloat(response, "duration"); duration != nil {
		rounded := int(*duration)
		media.DurationSeconds = &rounded
	}

	if originalURL := getString(response, "original_url"); originalURL != "" {
		sfc := isShortFormContent(response)
		media.ShortFormContent = &sfc
	}

	media.UploadedAt = parseUploadedAt(response)

	if playlistIndex := getInt(response, "playlist_index"); playlistIndex != nil {
		media.PlaylistIndex = playlistIndex
	} else {
		zero := 0
		media.PlaylistIndex = &zero
	}

	media.PredictedMediaFilepath = getString(response, "filename")

	switch getString(response, "availability") {
	case "subscriber_only", "premium_only":
		media.RequiresMembership = true
	}

	return media
}

func getString(response map[string]any, key string) string {
	if val, ok := response[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloat(response map[string]any, key string) *float64 {
	if val, ok := response[key]; ok && val != nil {
		if jn, ok := val.(json.Number); ok {
			if f, err := jn.Float64(); err == nil {
				return &f
			}
		}
		if f, ok := val.(float64); ok {
			return &f
		}
	}
	return nil
}

func getInt(response map[string]any, key string) *int {
	if val, ok := response[key]; ok && val != nil {
		if jn, ok := val.(json.Number); ok {
			if i, err := jn.Int64(); err == nil {
				result := int(i)
				return &result
			}
		}
		if f, ok := val.(float64); ok {
			i := int(f)
			return &i
		}
	}
	return nil
}

func isLivestream(response map[string]any) bool {
	liveStatus := getString(response, "live_status")
	return liveStatus != "" && liveStatus != "not_live"
}

func isShortFormContent(response map[string]any) bool {
	originalURL := getString(response, "original_url")
	if strings.Contains(originalURL, "/shorts/") {
		return true
	}

	duration := getFloat(response, "duration")
	aspectRatio := getFloat(response, "aspect_ratio")
	return duration != nil && aspectRatio != nil && *duration <= 180 && *aspectRatio <= 0.85
}

func parseUploadedAt(response map[string]any) *time.Time {
	if timestamp := getInt(response, "timestamp"); timestamp != nil {
		t := time.Unix(int64(*timestamp), 0).UTC()
		return &t
	}

	uploadDate := getString(response, "upload_date")
	if uploadDate == "" {
		return nil
	}

	parsedTime, err := ParseUploadDate(uploadDate)
	if err != nil {
		return nil
	}
	return &parsedTime
}

// ParseUploadDate parses a yt-dlp "YYYYMMDD" upload_date into midnight UTC.
func ParseUploadDate(uploadDate string) (time.Time, error) {
	if len(uploadDate) < 8 {
		return time.Time{}, fmt.Errorf("Invalid upload date: %s", uploadDate)
	}
	year, month, day := uploadDate[0:4], uploadDate[4:6], uploadDate[6:8]
	dt, err := time.Parse("2006-01-02T15:04:05Z", fmt.Sprintf("%s-%s-%sT00:00:00Z", year, month, day))
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid upload date: %s", uploadDate)
	}
	return dt, nil
}
