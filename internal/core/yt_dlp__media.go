package core

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// YtDlpMedia is an alias kept for the many existing core call sites; the
// type itself lives in internal/ytdlp alongside its response parsing.
type YtDlpMedia = ytdlp.Media

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

// YtDlpMediaIndexingOutputTemplate is ytdlp.IndexingOutputTemplate.
func YtDlpMediaIndexingOutputTemplate() string { return ytdlp.IndexingOutputTemplate() }

// YtDlpMediaResponseToStruct is ytdlp.ResponseToStruct.
func YtDlpMediaResponseToStruct(response map[string]any) *YtDlpMedia {
	return ytdlp.ResponseToStruct(response)
}

func ytDlpMediaGetString(response map[string]any, key string) string {
	if val, ok := response[key]; ok && val != nil {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
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
