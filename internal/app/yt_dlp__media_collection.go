package app

import (
	"context"
	"log/slog"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// MediaCollectionGetMediaAttributesForCollection/3
func (a *App) MediaCollectionGetMediaAttributesForCollection(ctx context.Context, url string, commandOpts store.KW, addlOpts store.KW) ([]*YtDlpMedia, error) {
	// ignore_no_formats_error is necessary because yt-dlp will error out if
	// the first video has not released yet (ie: is a premier). We don't care about
	// available formats since we're just getting the media details
	allCommandOpts := append(
		store.KW{store.Flag("simulate"), store.Flag("skip_download"), store.Flag("ignore_no_formats_error"), store.Flag("no_warnings")},
		commandOpts...,
	)

	useCookies := addlOpts.Bool("use_cookies")
	outputTemplate := YtDlpMediaIndexingOutputTemplate()
	outputFilepath, err := fsutil.GenerateTmpfile(a.Config.TmpfileDirectory, "json")
	if err != nil {
		return nil, err
	}

	fileListenerHandler, ok := addlOpts.Get("file_listener_handler")
	if ok && fileListenerHandler != nil {
		if handler, ok := fileListenerHandler.(func(string)); ok {
			handler(outputFilepath)
		}
	}

	runnerOpts := store.KW{store.Opt("output_filepath", outputFilepath), store.Opt("use_cookies", useCookies)}
	output, err := a.YtDlp.Run(ctx, url, "get_media_attributes_for_collection", allCommandOpts, outputTemplate, runnerOpts)
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
func (a *App) MediaCollectionGetSourceDetails(ctx context.Context, sourceURL string, commandOpts store.KW, addlOpts store.KW) (map[string]any, error) {
	// ignore_no_formats_error is necessary because yt-dlp will error out if
	// the first video has not released yet (ie: is a premier). We don't care about
	// available formats since we're just getting the source details
	defaultOpts := store.KW{
		store.Flag("simulate"),
		store.Flag("skip_download"),
		store.Flag("ignore_no_formats_error"),
		store.Opt("playlist_end", 1),
	}

	allCommandOpts := append(defaultOpts, commandOpts...)
	outputTemplate := "%(.{channel,channel_id,playlist_id,playlist_title,filename})j"

	output, err := a.YtDlp.Run(ctx, sourceURL, "get_source_details", allCommandOpts, outputTemplate, addlOpts)
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
func (a *App) MediaCollectionGetSourceMetadata(ctx context.Context, sourceURL string, commandOpts store.KW, addlOpts store.KW) (map[string]any, error) {
	// Validate that playlist_items is present
	if _, ok := commandOpts.Get("playlist_items"); !ok {
		slog.Error("playlist_items is required in commandOpts")
		return nil, ErrMissingPlaylistItems{}
	}

	allCommandOpts := append(store.KW{store.Flag("skip_download")}, commandOpts...)
	outputTemplate := "playlist:%()j"

	output, err := a.YtDlp.Run(ctx, sourceURL, "get_source_metadata", allCommandOpts, outputTemplate, addlOpts)
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
