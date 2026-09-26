package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// MediaDownloadWorker handles media download jobs.

const MediaDownloadWorkerName = "Pinchflat.Downloading.MediaDownloadWorker"

var mediaDownloadWorkerOpts = obanlite.WorkerOpts{
	Queue:    "media_fetching",
	Priority: 5,
	Unique: &obanlite.UniqueOpts{
		Period: obanlite.Infinity,
		States: []string{"available", "scheduled", "retryable", "executing"},
	},
	Tags: []string{"media_item", "media_fetching", "show_in_dashboard"},
}

// MediaDownloadWorkerKickoffWithTask/3
func (a *App) MediaDownloadWorkerKickoffWithTask(ctx context.Context, mediaItem *MediaItem, jobArgs Attrs, jobOpts KW) (*Task, error) {
	// Build job args: start with {id: mediaItem.id} and merge jobArgs
	args := Attrs{"id": mediaItem.ID}
	for k, v := range jobArgs {
		args[k] = v
	}

	// Create job spec
	spec := obanlite.JobSpec{
		Worker: MediaDownloadWorkerName,
		Args:   args,
	}

	// Handle job options (like priority)
	if priority, ok := jobOpts.Get("priority"); ok {
		if p, ok := priority.(int); ok {
			spec.Priority = &p
		}
	}

	// If schedule_in is specified, add it
	if schedIn, ok := jobOpts.Get("schedule_in"); ok {
		if s, ok := schedIn.(int); ok {
			spec.ScheduleIn = s
		}
	}

	// Create the job with task
	return a.TasksCreateJobWithTask(ctx, spec, mediaItem)
}

// MediaDownloadWorkerPerform/1
func (a *App) MediaDownloadWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	// Decode job args
	var args struct {
		ID             int64 `json:"id"`
		Force          bool  `json:"force"`
		QualityUpgrade bool  `json:"quality_upgrade?"`
	}

	if err := job.DecodeArgs(&args); err != nil {
		return err
	}

	// Fetch media item and run pre-download user script
	mediaItem, err := a.fetchAndRunPreventDownloadUserScript(ctx, args.ID)
	if err != nil {
		return nil // Errors from missing items should not retry
	}

	if mediaItem == nil {
		return nil // Missing item, don't retry
	}

	// Check if we should download
	shouldDownload := a.shouldDownloadMedia(ctx, mediaItem, args.Force, args.QualityUpgrade)
	if !shouldDownload {
		return nil
	}

	return a.downloadMediaAndScheduleJobs(ctx, mediaItem, args.QualityUpgrade, args.Force)
}

// fetchAndRunPreventDownloadUserScript runs the media_pre_download user script and sets prevent_download if it fails
func (a *App) fetchAndRunPreventDownloadUserScript(ctx context.Context, mediaItemID int64) (*MediaItem, error) {
	mediaItem, err := a.MediaGetMediaItem(ctx, mediaItemID)
	if err != nil {
		// Log and return nil for missing item (don't retry)
		slog.Info(fmt.Sprintf("%s discarded: media item %d not found", MediaDownloadWorkerName, mediaItemID))
		return nil, nil
	}

	// Run user script
	userScriptErr := a.UserScripts.Run(ctx, "media_pre_download", mediaItem)
	if userScriptErr != nil {
		// If non-zero exit, set prevent_download
		// We check if this looks like an exit code error
		if strings.Contains(userScriptErr.Error(), "1") || strings.Contains(userScriptErr.Error(), "exit") {
			_, updateErr := a.MediaUpdateMediaItem(ctx, mediaItem, Attrs{"prevent_download": true})
			if updateErr != nil {
				return nil, updateErr
			}
		}
	}

	// Preload source
	mediaItem, err = a.mediaPreloadSource(ctx, mediaItem)
	if err != nil {
		return nil, err
	}

	return mediaItem, nil
}

// shouldDownloadMedia checks if we should download based on various conditions
func (a *App) shouldDownloadMedia(ctx context.Context, mediaItem *MediaItem, shouldForce bool, isQualityUpgrade bool) bool {
	if isQualityUpgrade {
		// For quality upgrades, only check if source allows download and media isn't prevented
		return (mediaItem.Source.DownloadMedia && !mediaItem.PreventDownload) || shouldForce
	}

	// For normal downloads, additionally check if media item is pending
	isPending, err := a.MediaPendingDownload(ctx, mediaItem)
	if err != nil {
		return false
	}

	return (isPending && mediaItem.Source.DownloadMedia && !mediaItem.PreventDownload) || shouldForce
}

// downloadMediaAndScheduleJobs handles the actual download and subsequent tasks
func (a *App) downloadMediaAndScheduleJobs(ctx context.Context, mediaItem *MediaItem, isQualityUpgrade bool, shouldForce bool) error {
	// Determine overwrite behavior
	var overwriteBehavior string
	if shouldForce || isQualityUpgrade {
		overwriteBehavior = "force_overwrites"
	} else {
		overwriteBehavior = "no_force_overwrites"
	}

	overrideOpts := KW{Opt("overwrite_behaviour", overwriteBehavior)}

	// Download media
	result, downloadErr := a.MediaDownloaderDownloadForMediaItem(ctx, mediaItem, overrideOpts)

	// Success case: {:ok, downloaded_media_item}
	if downloadErr == nil && result != nil && !result.Recovered {
		// Update media item with file size and redownload timestamp
		fileSize := a.computeMediaFilesize(result.MediaItem)
		redownloadedAt := a.getRedownloadedAt(isQualityUpgrade)

		attrs := Attrs{}
		if fileSize != nil {
			attrs["media_size_bytes"] = *fileSize
		}
		if redownloadedAt != nil {
			attrs["media_redownloaded_at"] = *redownloadedAt
		}

		updatedMediaItem, updateErr := a.MediaUpdateMediaItem(ctx, result.MediaItem, attrs)
		if updateErr != nil {
			return updateErr
		}

		// Delete outdated files
		if delErr := FileSyncingDeleteOutdatedFiles(ctx, mediaItem, updatedMediaItem); delErr != nil {
			return delErr
		}

		// Run post-download user script
		_ = a.UserScripts.Run(ctx, "media_downloaded", updatedMediaItem)

		return nil
	}

	// Recovered case: {:recovered, _media_item, _message}
	if result != nil && result.Recovered {
		// Return an error to retry
		return errors.New("download_recovered")
	}

	// Error case
	if downloadErr == nil {
		return nil // No error
	}

	// Check if it's a MediaDownloaderError
	if mdErr, ok := downloadErr.(*MediaDownloaderError); ok {
		if mdErr.Reason == "unsuitable_for_download" {
			// This is non-retryable, return nil
			return nil
		}
	}

	// Handle action on error
	return a.actionOnError(downloadErr)
}

// computeMediaFilesize gets the file size of the downloaded media
func (a *App) computeMediaFilesize(mediaItem *MediaItem) *int64 {
	if mediaItem == nil || mediaItem.MediaFilepath == nil || *mediaItem.MediaFilepath == "" {
		return nil
	}

	stat, err := os.Stat(*mediaItem.MediaFilepath)
	if err != nil {
		return nil
	}

	size := stat.Size()
	return &size
}

// getRedownloadedAt returns the current time if isQualityUpgrade is true, nil otherwise
func (a *App) getRedownloadedAt(isQualityUpgrade bool) *time.Time {
	if isQualityUpgrade {
		now := time.Now().UTC()
		return &now
	}
	return nil
}

// actionOnError determines how to handle download errors
func (a *App) actionOnError(err error) error {
	nonRetryableErrors := []string{
		"Video unavailable",
		"Sign in to confirm",
		"This video is available to this channel's members",
	}

	errStr := err.Error()
	for _, pattern := range nonRetryableErrors {
		if strings.Contains(errStr, pattern) {
			slog.Error(fmt.Sprintf("yt-dlp download will not be retried: %v", err))
			// Return nil to mark as non-retryable but completed
			return nil
		}
	}

	// Return an error to retry
	return errors.New("download_failed")
}
