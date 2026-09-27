package core

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

// MediaDownloader is the integration layer for downloading media.

// MediaDownloaderResult is the non-error outcome of download_for_media_item/2:
// {:ok, media_item} (Recovered false) or {:recovered, media_item, message}.
type MediaDownloaderResult struct {
	MediaItem *MediaItem
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
func (a *App) MediaDownloaderDownloadForMediaItem(ctx context.Context, mediaItem *MediaItem, overrideOpts KW) (*MediaDownloaderResult, error) {
	result, err := mediaDownloaderAttemptDownloadAndUpdateForMediaItem(ctx, a, mediaItem, overrideOpts)

	if err != nil {
		// This is an error result, return it as is
		return nil, err
	}

	// This is a successful result (either :ok or :recovered)
	return result, nil
}

// --- Private helpers ---

func mediaDownloaderAttemptDownloadAndUpdateForMediaItem(ctx context.Context, a *App, mediaItem *MediaItem, overrideOpts KW) (*MediaDownloaderResult, error) {
	outputFilepath, _ := fsutil.GenerateTmpfile(a.Config.TmpfileDirectory, "json")
	mediaWithPreloads, _ := a.PreloadMediaItemFull(ctx, mediaItem)

	parsedJSON, dlErr := mediaDownloaderDownloadWithOptions(ctx, a, mediaItem.OriginalURL, mediaWithPreloads, outputFilepath, overrideOpts)

	if dlErr == nil {
		// Success
		updatedMediaItem, err := mediaDownloaderUpdateMediaItemFromParsedJSON(ctx, a, mediaWithPreloads, parsedJSON)
		if err != nil {
			// If update fails, clear last_error before returning the update error
			return nil, err
		}
		// Clear last_error on success
		updatedMediaItem2, _ := a.MediaUpdateMediaItem(ctx, updatedMediaItem, Attrs{"last_error": nil})
		return &MediaDownloaderResult{MediaItem: updatedMediaItem2, Recovered: false}, nil
	}

	// Check if it's an unsuitable error
	if mediaDownloaderIsUnsuitable(dlErr) {
		message := fmt.Sprintf("Media item #%d isn't suitable for download yet. May be an active or processing live stream", mediaWithPreloads.ID)
		slog.Warn(message)
		// Update last_error
		a.MediaUpdateMediaItem(ctx, mediaWithPreloads, Attrs{"last_error": message})
		return nil, &MediaDownloaderError{Reason: "unsuitable_for_download", Message: message}
	}

	// Check if it's a command error (yt-dlp error)
	cmdErr, isCommandError := dlErr.(*fsutil.CommandError)
	if isCommandError {
		errMsg := cmdErr.Output
		slog.Error(fmt.Sprintf("yt-dlp download error for media item #%d: %v", mediaWithPreloads.ID, dlErr))

		if mediaDownloaderIsRecoverableError(errMsg) {
			return mediaDownloaderAttemptRecoveryFromError(ctx, a, mediaWithPreloads, outputFilepath, errMsg)
		}

		// Update last_error and return error
		a.MediaUpdateMediaItem(ctx, mediaWithPreloads, Attrs{"last_error": errMsg})
		return nil, &MediaDownloaderError{Reason: "download_failed", Message: errMsg}
	}

	// Unknown error
	slog.Error(fmt.Sprintf("Unknown error downloading media item #%d: %v", mediaWithPreloads.ID, dlErr))
	errMsg := fmt.Sprintf("Unknown error: %v", dlErr)
	a.MediaUpdateMediaItem(ctx, mediaWithPreloads, Attrs{"last_error": errMsg})
	return nil, &MediaDownloaderError{Reason: "unknown", Message: errMsg}
}

func mediaDownloaderAttemptRecoveryFromError(ctx context.Context, a *App, mediaWithPreloads *MediaItem, outputFilepath string, errorMessage string) (*MediaDownloaderResult, error) {
	contents, err := os.ReadFile(outputFilepath)
	if err != nil {
		slog.Error(fmt.Sprintf("Unable to recover error for media item #%d: %v", mediaWithPreloads.ID, err))
		a.MediaUpdateMediaItem(ctx, mediaWithPreloads, Attrs{"last_error": errorMessage})
		return nil, &MediaDownloaderError{Reason: "unrecoverable", Message: errorMessage}
	}

	var parsedJSON map[string]any
	err = DecodeJSON(contents, &parsedJSON)
	if err != nil {
		slog.Error(fmt.Sprintf("Unable to recover error for media item #%d: %v", mediaWithPreloads.ID, err))
		a.MediaUpdateMediaItem(ctx, mediaWithPreloads, Attrs{"last_error": errorMessage})
		return nil, &MediaDownloaderError{Reason: "unrecoverable", Message: errorMessage}
	}

	slog.Info(fmt.Sprintf("Recovery from yt-dlp error seems possible. Updating media item #%d with parsed JSON from partial download attempt. Full download will be re-attemted in future anyway", mediaWithPreloads.ID))

	updatedMediaItem, err := mediaDownloaderUpdateMediaItemFromParsedJSON(ctx, a, mediaWithPreloads, parsedJSON)
	if err != nil {
		a.MediaUpdateMediaItem(ctx, mediaWithPreloads, Attrs{"last_error": errorMessage})
		return nil, &MediaDownloaderError{Reason: "unrecoverable", Message: errorMessage}
	}

	// Update last_error with the warning message
	updatedMediaItem2, _ := a.MediaUpdateMediaItem(ctx, updatedMediaItem, Attrs{"last_error": errorMessage})
	return &MediaDownloaderResult{MediaItem: updatedMediaItem2, Recovered: true, Message: errorMessage}, nil
}

func mediaDownloaderUpdateMediaItemFromParsedJSON(ctx context.Context, a *App, mediaWithPreloads *MediaItem, parsedJSON map[string]any) (*MediaItem, error) {
	parsedAttrs, _ := MetadataParserParseForMediaItem(parsedJSON)

	// Merge in additional attributes
	parsedAttrs["media_downloaded_at"] = db.Now()
	parsedAttrs["culled_at"] = nil

	nfoFilepath := mediaDownloaderDetermineNfoFilepath(parsedJSON, mediaWithPreloads.Source.MediaProfile.DownloadNfo)
	parsedAttrs["nfo_filepath"] = nfoFilepath

	// Handle metadata
	metadataFilepath, _ := a.MetadataFileHelpersCompressAndStoreMetadataFor(ctx, mediaWithPreloads, parsedJSON)
	thumbnailFilepath, _ := a.MetadataFileHelpersDownloadAndStoreThumbnailFor(ctx, mediaWithPreloads)

	metadataMap := map[string]any{
		"metadata_filepath":  metadataFilepath,
		"thumbnail_filepath": thumbnailFilepath,
	}
	parsedAttrs["metadata"] = metadataMap

	return a.MediaUpdateMediaItem(ctx, mediaWithPreloads, parsedAttrs)
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
	return Ptr(result)
}

func mediaDownloaderDownloadWithOptions(ctx context.Context, a *App, url string, itemWithPreloads *MediaItem, outputFilepath string, overrideOpts KW) (map[string]any, error) {
	opts, _ := a.DownloadOptionBuilderBuild(ctx, itemWithPreloads, overrideOpts)
	forceUseCookies := overrideOpts.GetOr("force_use_cookies", false).(bool)
	sourceUsesCookies := SourcesUseCookies(itemWithPreloads.Source, "downloading")
	shouldUseCookies := forceUseCookies || sourceUsesCookies

	runnerOpts := KW{Opt("output_filepath", outputFilepath), Opt("use_cookies", shouldUseCookies)}

	// Check downloadable status
	statusStr, statusErr := a.YtDlpMediaGetDownloadableStatus(ctx, url, runnerOpts)
	if statusErr != nil {
		if !shouldUseCookies {
			// Try with cookies if it might help
			return mediaDownloaderMaybeRetryWithCookies(ctx, a, url, itemWithPreloads, outputFilepath, overrideOpts, statusErr)
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
		return mediaDownloaderMaybeRetryWithCookies(ctx, a, url, itemWithPreloads, outputFilepath, overrideOpts, err)
	}

	return nil, err
}

func mediaDownloaderMaybeRetryWithCookies(ctx context.Context, a *App, url string, itemWithPreloads *MediaItem, outputFilepath string, overrideOpts KW, err error) (map[string]any, error) {
	source := itemWithPreloads.Source
	cmdErr, isCommandError := err.(*fsutil.CommandError)
	if !isCommandError {
		return nil, err
	}

	messageStr := cmdErr.Output
	messageContainsCookieError := mediaDownloaderIsRecoverableCookieError(messageStr)

	if SourcesUseCookies(source, "error_recovery") && messageContainsCookieError {
		// Retry with cookies
		newOverrideOpts := make(KW, len(overrideOpts))
		copy(newOverrideOpts, overrideOpts)
		newOverrideOpts = append(newOverrideOpts, Opt("force_use_cookies", true))
		return mediaDownloaderDownloadWithOptions(ctx, a, url, itemWithPreloads, outputFilepath, newOverrideOpts)
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
