package core

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// YtDlpMedia represents a piece of media parsed from yt-dlp's output.
// Not a database table, just a struct used for parsing yt-dlp responses.
type YtDlpMedia struct {
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
}

// YtDlpMediaDownload/3
func (a *App) YtDlpMediaDownload(ctx context.Context, url string, commandOpts KW, addlOpts KW) (map[string]any, error) {
	allCommandOpts := append(KW{Flag("no_simulate")}, commandOpts...)

	output, err := a.YtDlp.Run(ctx, url, "download", allCommandOpts, "after_move:%()j", addlOpts)
	if err != nil {
		return nil, err
	}

	result := map[string]any{}
	if err := DecodeJSON([]byte(output), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// YtDlpMediaGetDownloadableStatus/2
func (a *App) YtDlpMediaGetDownloadableStatus(ctx context.Context, url string, addlOpts KW) (string, error) {
	commandOpts := KW{Flag("simulate"), Flag("skip_download")}

	output, err := a.YtDlp.Run(ctx, url, "get_downloadable_status", commandOpts, "%(.{live_status})j", addlOpts)
	if err != nil {
		return "", err
	}

	parsed := map[string]any{}
	if err := DecodeJSON([]byte(output), &parsed); err != nil {
		return "", err
	}

	return ytDlpMediaParseDownloadableStatus(parsed)
}

// YtDlpMediaDownloadThumbnail/3
func (a *App) YtDlpMediaDownloadThumbnail(ctx context.Context, url string, commandOpts KW, addlOpts KW) (string, error) {
	allCommandOpts := append(
		KW{Flag("no_simulate"), Flag("skip_download"), Flag("write_thumbnail"), Opt("convert_thumbnail", "jpg")},
		commandOpts...,
	)

	output, err := a.YtDlp.Run(ctx, url, "download_thumbnail", allCommandOpts, "after_move:%()j", addlOpts)
	if err != nil {
		return "", err
	}

	return output, nil
}

// YtDlpMediaGetMediaAttributes/3
func (a *App) YtDlpMediaGetMediaAttributes(ctx context.Context, url string, commandOpts KW, addlOpts KW) (*YtDlpMedia, error) {
	allCommandOpts := append(KW{Flag("simulate"), Flag("skip_download")}, commandOpts...)
	outputTemplate := YtDlpMediaIndexingOutputTemplate()

	output, err := a.YtDlp.Run(ctx, url, "get_media_attributes", allCommandOpts, outputTemplate, addlOpts)
	if err != nil {
		return nil, err
	}

	parsed := map[string]any{}
	if err := DecodeJSON([]byte(output), &parsed); err != nil {
		return nil, err
	}

	return YtDlpMediaResponseToStruct(parsed), nil
}

// YtDlpMediaIndexingOutputTemplate/0
func YtDlpMediaIndexingOutputTemplate() string {
	return "%(.{id,title,live_status,original_url,description,aspect_ratio,duration,upload_date,timestamp,playlist_index,filename})j"
}

// YtDlpMediaResponseToStruct/1
func YtDlpMediaResponseToStruct(response map[string]any) *YtDlpMedia {
	media := &YtDlpMedia{
		MediaID:     ytDlpMediaGetString(response, "id"),
		Title:       ytDlpMediaGetString(response, "title"),
		Description: ytDlpMediaGetString(response, "description"),
		OriginalURL: ytDlpMediaGetString(response, "original_url"),
		Livestream:  ytDlpMediaIsLivestream(response),
	}

	// Handle duration_seconds
	if duration := ytDlpMediaGetFloat(response, "duration"); duration != nil {
		rounded := int(*duration)
		media.DurationSeconds = &rounded
	}

	// Handle short_form_content
	if originalURL := ytDlpMediaGetString(response, "original_url"); originalURL != "" {
		sfc := ytDlpMediaShortFormContent(response)
		media.ShortFormContent = &sfc
	}

	// Handle uploaded_at
	media.UploadedAt = ytDlpMediaParseUploadedAt(response)

	// Handle playlist_index
	if playlistIndex := ytDlpMediaGetInt(response, "playlist_index"); playlistIndex != nil {
		media.PlaylistIndex = playlistIndex
	} else {
		zero := 0
		media.PlaylistIndex = &zero
	}

	media.PredictedMediaFilepath = ytDlpMediaGetString(response, "filename")

	return media
}

func ytDlpMediaGetString(response map[string]any, key string) string {
	if val, ok := response[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func ytDlpMediaGetFloat(response map[string]any, key string) *float64 {
	if val, ok := response[key]; ok && val != nil {
		// Handle json.Number type
		if jn, ok := val.(json.Number); ok {
			f, err := jn.Float64()
			if err == nil {
				return &f
			}
		}
		// Handle float64 type
		if f, ok := val.(float64); ok {
			return &f
		}
	}
	return nil
}

func ytDlpMediaGetInt(response map[string]any, key string) *int {
	if val, ok := response[key]; ok && val != nil {
		// Handle json.Number type
		if jn, ok := val.(json.Number); ok {
			i, err := jn.Int64()
			if err == nil {
				result := int(i)
				return &result
			}
		}
		// Handle float64 type (JSON numbers come as float64)
		if f, ok := val.(float64); ok {
			i := int(f)
			return &i
		}
	}
	return nil
}

func ytDlpMediaIsLivestream(response map[string]any) bool {
	liveStatus := ytDlpMediaGetString(response, "live_status")
	return liveStatus != "" && liveStatus != "not_live"
}

func ytDlpMediaShortFormContent(response map[string]any) bool {
	originalURL := ytDlpMediaGetString(response, "original_url")
	if strings.Contains(originalURL, "/shorts/") {
		return true
	}

	// Heuristic: duration <= 180 and aspect_ratio <= 0.85
	duration := ytDlpMediaGetFloat(response, "duration")
	aspectRatio := ytDlpMediaGetFloat(response, "aspect_ratio")

	if duration != nil && aspectRatio != nil && *duration <= 180 && *aspectRatio <= 0.85 {
		return true
	}

	return false
}

func ytDlpMediaParseUploadedAt(response map[string]any) *time.Time {
	// Try to parse from timestamp first
	if timestamp := ytDlpMediaGetInt(response, "timestamp"); timestamp != nil {
		t := time.Unix(int64(*timestamp), 0).UTC()
		return &t
	}

	// Fall back to upload_date
	uploadDate := ytDlpMediaGetString(response, "upload_date")
	if uploadDate == "" {
		return nil
	}

	parsedTime, err := MetadataFileHelpersParseUploadDate(uploadDate)
	if err != nil {
		return nil
	}

	return &parsedTime
}

func ytDlpMediaParseDownloadableStatus(response map[string]any) (string, error) {
	liveStatus := ytDlpMediaGetString(response, "live_status")

	switch liveStatus {
	case "is_live", "is_upcoming", "post_live":
		return "ignorable", nil
	case "was_live", "not_live":
		return "downloadable", nil
	case "":
		// nil live_status is treated as downloadable
		return "downloadable", nil
	default:
		return "", ErrUnknownLiveStatus{Status: liveStatus}
	}
}

// ErrUnknownLiveStatus is returned when we get an unknown live status
type ErrUnknownLiveStatus struct {
	Status string
}

func (e ErrUnknownLiveStatus) Error() string {
	return "Unknown live status: " + e.Status
}
