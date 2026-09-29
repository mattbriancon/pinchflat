package app

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// YtDlpMedia is an alias kept for the many existing core call sites; the
// type itself lives in internal/ytdlp alongside its response parsing.
type YtDlpMedia = ytdlp.Media

// YtDlpMediaDownload/3
func (a *App) YtDlpMediaDownload(ctx context.Context, url string, args ytdlp.Args, opts ytdlp.CallOptions) (map[string]any, error) {
	allArgs := ytdlp.Args{}.Flag("no_simulate")
	allArgs = append(allArgs, args...)

	output, err := a.YtDlp.Run(ctx, url, "download", allArgs, "after_move:%()j", opts)
	if err != nil {
		return nil, err
	}

	result := map[string]any{}
	if err := store.DecodeJSON([]byte(output), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// YtDlpMediaGetDownloadableStatus/2
func (a *App) YtDlpMediaGetDownloadableStatus(ctx context.Context, url string, opts ytdlp.CallOptions) (string, error) {
	args := ytdlp.Args{}.Flag("simulate").Flag("skip_download")

	output, err := a.YtDlp.Run(ctx, url, "get_downloadable_status", args, "%(.{live_status})j", opts)
	if err != nil {
		return "", err
	}

	parsed := map[string]any{}
	if err := store.DecodeJSON([]byte(output), &parsed); err != nil {
		return "", err
	}

	return ytDlpMediaParseDownloadableStatus(parsed)
}

// YtDlpMediaDownloadThumbnail/3
func (a *App) YtDlpMediaDownloadThumbnail(ctx context.Context, url string, args ytdlp.Args, opts ytdlp.CallOptions) (string, error) {
	allArgs := ytdlp.Args{}.Flag("no_simulate").Flag("skip_download").Flag("write_thumbnail").Opt("convert_thumbnail", "jpg")
	allArgs = append(allArgs, args...)

	output, err := a.YtDlp.Run(ctx, url, "download_thumbnail", allArgs, "after_move:%()j", opts)
	if err != nil {
		return "", err
	}

	return output, nil
}

// YtDlpMediaGetMediaAttributes/3
func (a *App) YtDlpMediaGetMediaAttributes(ctx context.Context, url string, args ytdlp.Args, opts ytdlp.CallOptions) (*YtDlpMedia, error) {
	allArgs := ytdlp.Args{}.Flag("simulate").Flag("skip_download")
	allArgs = append(allArgs, args...)
	outputTemplate := YtDlpMediaIndexingOutputTemplate()

	output, err := a.YtDlp.Run(ctx, url, "get_media_attributes", allArgs, outputTemplate, opts)
	if err != nil {
		return nil, err
	}

	parsed := map[string]any{}
	if err := store.DecodeJSON([]byte(output), &parsed); err != nil {
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
