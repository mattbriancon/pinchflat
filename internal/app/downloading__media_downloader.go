package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/mattbriancon/pinchflat/internal/cmdrun"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// MediaDownloader is the integration layer for downloading media.

// MediaDownloaderResult is the non-error outcome of download_for_media_item/2:
// {:ok, media_item} (Recovered false) or {:recovered, media_item, message}.
type MediaDownloaderResult struct {
	MediaItem *store.MediaItem
	Recovered bool
	Message   string
}

// MediaDownloaderError is {:error, error_atom, message}. Reason is the atom
// name: "unsuitable_for_download", "download_failed", "unrecoverable" or "unknown".
type MediaDownloaderError struct {
	Reason  string
	Message string
}

func (e *MediaDownloaderError) Error() string { return e.Reason + ": " + e.Message }

// download_for_media_item/2
func (a *App) MediaDownloaderDownloadForMediaItem(ctx context.Context, mediaItem *store.MediaItem, overrides DownloadOverrides) (*MediaDownloaderResult, error) {
	result, err := mediaDownloaderAttemptDownloadAndUpdateForMediaItem(ctx, a, mediaItem, overrides)

	if err != nil {
		// This is an error result, return it as is
		return nil, err
	}

	// This is a successful result (either :ok or :recovered)
	return result, nil
}

// --- Private helpers ---

func mediaDownloaderAttemptDownloadAndUpdateForMediaItem(ctx context.Context, a *App, mediaItem *store.MediaItem, overrides DownloadOverrides) (*MediaDownloaderResult, error) {
	outputFilepath, _ := fsutil.GenerateTmpfile(a.Config.TmpfileDirectory, "json")
	mediaWithPreloads, _ := a.PreloadMediaItemFull(ctx, mediaItem)

	parsedJSON, dlErr := mediaDownloaderDownloadWithOptions(ctx, a, mediaItem.OriginalURL, mediaWithPreloads, outputFilepath, overrides)

	if dlErr == nil {
		// Success
		updatedMediaItem, err := mediaDownloaderUpdateMediaItemFromParsedJSON(ctx, a, mediaWithPreloads, parsedJSON)
		if err != nil {
			// If update fails, clear last_error before returning the update error
			return nil, err
		}
		// Clear last_error on success
		updatedMediaItem2, _ := a.UpdateMediaItem(ctx, updatedMediaItem, store.MediaItemParams{Clear: store.ClearLastError})
		return &MediaDownloaderResult{MediaItem: updatedMediaItem2, Recovered: false}, nil
	}

	// Check if it's an unsuitable error
	if mediaDownloaderIsUnsuitable(dlErr) {
		message := fmt.Sprintf("Media item #%d isn't suitable for download yet. May be an active or processing live stream", mediaWithPreloads.ID)
		slog.Warn(message)
		// Update last_error
		a.UpdateMediaItem(ctx, mediaWithPreloads, store.MediaItemParams{LastError: &message})
		return nil, &MediaDownloaderError{Reason: "unsuitable_for_download", Message: message}
	}

	// Check if it's a command error (yt-dlp error)
	cmdErr, isCommandError := dlErr.(*cmdrun.Error)
	if isCommandError {
		errMsg := cmdErr.Output
		slog.Error(fmt.Sprintf("yt-dlp download error for media item #%d: %v", mediaWithPreloads.ID, dlErr))

		if mediaDownloaderIsRecoverableError(errMsg) {
			return mediaDownloaderAttemptRecoveryFromError(ctx, a, mediaWithPreloads, outputFilepath, errMsg)
		}

		// Update last_error and return error
		a.UpdateMediaItem(ctx, mediaWithPreloads, store.MediaItemParams{LastError: &errMsg})
		return nil, &MediaDownloaderError{Reason: "download_failed", Message: errMsg}
	}

	// Unknown error
	slog.Error(fmt.Sprintf("Unknown error downloading media item #%d: %v", mediaWithPreloads.ID, dlErr))
	errMsg := fmt.Sprintf("Unknown error: %v", dlErr)
	a.UpdateMediaItem(ctx, mediaWithPreloads, store.MediaItemParams{LastError: &errMsg})
	return nil, &MediaDownloaderError{Reason: "unknown", Message: errMsg}
}

func mediaDownloaderAttemptRecoveryFromError(ctx context.Context, a *App, mediaWithPreloads *store.MediaItem, outputFilepath string, errorMessage string) (*MediaDownloaderResult, error) {
	contents, err := os.ReadFile(outputFilepath)
	if err != nil {
		slog.Error(fmt.Sprintf("Unable to recover error for media item #%d: %v", mediaWithPreloads.ID, err))
		a.UpdateMediaItem(ctx, mediaWithPreloads, store.MediaItemParams{LastError: &errorMessage})
		return nil, &MediaDownloaderError{Reason: "unrecoverable", Message: errorMessage}
	}

	var parsedJSON map[string]any
	err = store.DecodeJSON(contents, &parsedJSON)
	if err != nil {
		slog.Error(fmt.Sprintf("Unable to recover error for media item #%d: %v", mediaWithPreloads.ID, err))
		a.UpdateMediaItem(ctx, mediaWithPreloads, store.MediaItemParams{LastError: &errorMessage})
		return nil, &MediaDownloaderError{Reason: "unrecoverable", Message: errorMessage}
	}

	slog.Info(fmt.Sprintf("Recovery from yt-dlp error seems possible. Updating media item #%d with parsed JSON from partial download attempt. Full download will be re-attemted in future anyway", mediaWithPreloads.ID))

	updatedMediaItem, err := mediaDownloaderUpdateMediaItemFromParsedJSON(ctx, a, mediaWithPreloads, parsedJSON)
	if err != nil {
		a.UpdateMediaItem(ctx, mediaWithPreloads, store.MediaItemParams{LastError: &errorMessage})
		return nil, &MediaDownloaderError{Reason: "unrecoverable", Message: errorMessage}
	}

	// Update last_error with the warning message
	updatedMediaItem2, _ := a.UpdateMediaItem(ctx, updatedMediaItem, store.MediaItemParams{LastError: &errorMessage})
	return &MediaDownloaderResult{MediaItem: updatedMediaItem2, Recovered: true, Message: errorMessage}, nil
}

func mediaDownloaderUpdateMediaItemFromParsedJSON(ctx context.Context, a *App, mediaWithPreloads *store.MediaItem, parsedJSON map[string]any) (*store.MediaItem, error) {
	params, _ := MetadataParserParseForMediaItem(parsedJSON)

	// Merge in additional attributes
	params.MediaDownloadedAt = store.Ptr(time.Now().UTC())
	params.Clear |= store.ClearCulledAt

	params.NfoFilepath = mediaDownloaderDetermineNfoFilepath(parsedJSON, mediaWithPreloads.Source.MediaProfile.DownloadNfo)
	if params.NfoFilepath == nil {
		params.Clear |= store.ClearNfoFilepath
	}

	// Handle metadata
	metadataFilepath, _ := a.MetadataFileHelpersCompressAndStoreMetadataFor(ctx, mediaWithPreloads, parsedJSON)
	thumbnailFilepath, _ := a.MetadataFileHelpersDownloadAndStoreThumbnailFor(ctx, mediaWithPreloads)
	if thumbnailFilepath == nil {
		thumbnailFilepath = store.Ptr("")
	}
	params.Metadata = &store.MediaMetadataParams{
		MetadataFilepath:  &metadataFilepath,
		ThumbnailFilepath: thumbnailFilepath,
	}

	return a.UpdateMediaItem(ctx, mediaWithPreloads, params)
}

func mediaDownloaderDetermineNfoFilepath(parsedJSON map[string]any, downloadNfo bool) *string {
	if !downloadNfo {
		return nil
	}

	filepath, ok := parsedJSON["filepath"].(string)
	if !ok {
		return nil
	}

	// Remove the extension and add .nfo
	ext := getExtension(filepath)
	nfoFilepath := strings.TrimSuffix(filepath, "."+ext) + ".nfo"

	result, _ := NfoBuilderBuildAndStoreForMediaItem(nfoFilepath, parsedJSON)
	return store.Ptr(result)
}

func mediaDownloaderDownloadWithOptions(ctx context.Context, a *App, url string, itemWithPreloads *store.MediaItem, outputFilepath string, overrides DownloadOverrides) (map[string]any, error) {
	opts, _ := a.DownloadOptionBuilderBuild(ctx, itemWithPreloads, overrides)
	sourceUsesCookies := store.UseCookies(itemWithPreloads.Source, "downloading")
	shouldUseCookies := overrides.ForceUseCookies || sourceUsesCookies

	runnerOpts := ytdlp.CallOptions{OutputFilepath: outputFilepath, UseCookies: shouldUseCookies}

	// Check downloadable status
	statusStr, statusErr := a.YtDlpMediaGetDownloadableStatus(ctx, url, runnerOpts)
	if statusErr != nil {
		if !shouldUseCookies {
			// Try with cookies if it might help
			return mediaDownloaderMaybeRetryWithCookies(ctx, a, url, itemWithPreloads, outputFilepath, overrides, statusErr)
		}
		return nil, statusErr
	}

	// Check if downloadable
	if statusStr == "ignorable" {
		return nil, &MediaDownloaderError{Reason: "unsuitable_for_download", Message: "unsuitable for download"}
	}

	// Proceed with download
	result, err := a.YtDlpMediaDownload(ctx, url, opts, runnerOpts)
	if err == nil {
		return result, nil
	}

	// If there was an error and we're not using cookies, maybe retry with cookies
	if !shouldUseCookies {
		return mediaDownloaderMaybeRetryWithCookies(ctx, a, url, itemWithPreloads, outputFilepath, overrides, err)
	}

	return nil, err
}

func mediaDownloaderMaybeRetryWithCookies(ctx context.Context, a *App, url string, itemWithPreloads *store.MediaItem, outputFilepath string, overrides DownloadOverrides, err error) (map[string]any, error) {
	source := itemWithPreloads.Source
	cmdErr, isCommandError := err.(*cmdrun.Error)
	if !isCommandError {
		return nil, err
	}

	messageStr := cmdErr.Output
	messageContainsCookieError := mediaDownloaderIsRecoverableCookieError(messageStr)

	if store.UseCookies(source, "error_recovery") && messageContainsCookieError {
		// Retry with cookies
		overrides.ForceUseCookies = true
		return mediaDownloaderDownloadWithOptions(ctx, a, url, itemWithPreloads, outputFilepath, overrides)
	}

	return nil, err
}

func mediaDownloaderIsUnsuitable(err error) bool {
	mediaErr, ok := err.(*MediaDownloaderError)
	if !ok {
		return false
	}
	return mediaErr.Reason == "unsuitable_for_download"
}

func mediaDownloaderIsRecoverableError(message string) bool {
	recoverable := []string{
		"Unable to communicate with SponsorBlock",
	}
	for _, err := range recoverable {
		if strings.Contains(message, err) {
			return true
		}
	}
	return false
}

func mediaDownloaderIsRecoverableCookieError(message string) bool {
	cookieErrors := []string{
		"Sign in to confirm",
		"This video is available to this channel's members",
	}
	for _, err := range cookieErrors {
		if strings.Contains(message, err) {
			return true
		}
	}
	return false
}

func getExtension(filepath string) string {
	lastDot := strings.LastIndex(filepath, ".")
	if lastDot == -1 {
		return ""
	}
	return filepath[lastDot+1:]
}
