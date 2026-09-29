package app

import (
	"context"
	"log/slog"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// MediaCollectionGetMediaAttributesForCollection/3
func (a *App) MediaCollectionGetMediaAttributesForCollection(ctx context.Context, url string, args ytdlp.Args, opts ytdlp.CallOptions, fileListener func(outputFilepath string)) ([]*YtDlpMedia, error) {
	// ignore_no_formats_error is necessary because yt-dlp will error out if
	// the first video has not released yet (ie: is a premier). We don't care about
	// available formats since we're just getting the media details
	allArgs := ytdlp.Args{}.Flag("simulate").Flag("skip_download").Flag("ignore_no_formats_error").Flag("no_warnings")
	allArgs = append(allArgs, args...)

	// The output file is created here so fileListener can start following it
	// before yt-dlp starts writing.
	outputFilepath, err := fsutil.GenerateTmpfile(a.Config.TmpfileDirectory, "json")
	if err != nil {
		return nil, err
	}
	if fileListener != nil {
		fileListener(outputFilepath)
	}
	opts.OutputFilepath = outputFilepath

	output, err := a.YtDlp.Run(ctx, url, "get_media_attributes_for_collection", allArgs, YtDlpMediaIndexingOutputTemplate(), opts)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(output, "\n")
	result := []*YtDlpMedia{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parsed := map[string]any{}
		if err := store.DecodeJSON([]byte(line), &parsed); err != nil {
			// Gracefully skip invalid JSON lines
			continue
		}

		result = append(result, YtDlpMediaResponseToStruct(parsed))
	}

	return result, nil
}

// MediaCollectionGetSourceDetails/3
func (a *App) MediaCollectionGetSourceDetails(ctx context.Context, sourceURL string, args ytdlp.Args, opts ytdlp.CallOptions) (map[string]any, error) {
	// ignore_no_formats_error is necessary because yt-dlp will error out if
	// the first video has not released yet (ie: is a premier). We don't care about
	// available formats since we're just getting the source details
	allArgs := ytdlp.Args{}.Flag("simulate").Flag("skip_download").Flag("ignore_no_formats_error").Opt("playlist_end", 1)
	allArgs = append(allArgs, args...)
	outputTemplate := "%(.{channel,channel_id,playlist_id,playlist_title,filename})j"

	output, err := a.YtDlp.Run(ctx, sourceURL, "get_source_details", allArgs, outputTemplate, opts)
	if err != nil {
		return nil, err
	}

	parsed := map[string]any{}
	if err := store.DecodeJSON([]byte(output), &parsed); err != nil {
		return nil, &ErrJSONDecode{Message: "Error decoding JSON response"}
	}

	return mediaCollectionFormatSourceDetails(parsed), nil
}

// MediaCollectionGetSourceMetadata/3
func (a *App) MediaCollectionGetSourceMetadata(ctx context.Context, sourceURL string, args ytdlp.Args, opts ytdlp.CallOptions) (map[string]any, error) {
	// Validate that playlist_items is present
	if _, ok := args.Get("playlist_items"); !ok {
		slog.Error("playlist_items is required in commandOpts")
		return nil, ErrMissingPlaylistItems{}
	}

	allArgs := ytdlp.Args{}.Flag("skip_download")
	allArgs = append(allArgs, args...)
	outputTemplate := "playlist:%()j"

	output, err := a.YtDlp.Run(ctx, sourceURL, "get_source_metadata", allArgs, outputTemplate, opts)
	if err != nil {
		return nil, err
	}

	parsed := map[string]any{}
	if err := store.DecodeJSON([]byte(output), &parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

func mediaCollectionFormatSourceDetails(response map[string]any) map[string]any {
	return map[string]any{
		"channel_id":    response["channel_id"],
		"channel_name":  response["channel"],
		"playlist_id":   response["playlist_id"],
		"playlist_name": response["playlist_title"],
		"filepath":      response["filename"],
	}
}

// ErrJSONDecode is returned when JSON decoding fails
type ErrJSONDecode struct {
	Message string
}

func (e *ErrJSONDecode) Error() string {
	return e.Message
}

// ErrMissingPlaylistItems is returned when playlist_items is not provided
type ErrMissingPlaylistItems struct{}

func (e ErrMissingPlaylistItems) Error() string {
	return "playlist_items is required in commandOpts"
}
