package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// SlowIndexingHelpers provides methods for performing slow indexing tasks.

// SlowIndexingHelpersKickoffIndexingTask/3
func (a *App) SlowIndexingHelpersKickoffIndexingTask(ctx context.Context, source *store.Source, jobArgs store.Attrs, jobOpts store.KW) (*store.Task, error) {
	jobOffsetSeconds := 0
	force := false
	if jobArgs != nil {
		if f, ok := jobArgs["force"]; ok {
			force, _ = f.(bool)
		}
	}

	if !force {
		jobOffsetSeconds = slowIndexingHelpersCalculateJobOffsetSeconds(source)
	}

	// Delete pending tasks for the source
	if err := a.DeletePendingTasksFor(ctx, source, stringPtr("MediaCollectionIndexingWorker"), store.KW{store.Opt("include_executing", true)}); err != nil {
		return nil, err
	}

	// Prepare job arguments
	args := store.Attrs{"id": source.ID}
	if jobArgs != nil {
		for k, v := range jobArgs {
			args[k] = v
		}
	}

	// Create job spec and task
	spec := obanlite.JobSpec{
		Worker:     MediaCollectionIndexingWorkerName,
		Args:       args,
		ScheduleIn: jobOffsetSeconds,
	}

	// Apply job options to spec
	if maxAttempts, ok := jobOpts.Get("max_attempts"); ok {
		spec.MaxAttempts = maxAttempts.(int)
	}

	return a.CreateJobWithTask(ctx, spec, source)
}

// SlowIndexingHelpersDeleteIndexingTasks/2
func (a *App) SlowIndexingHelpersDeleteIndexingTasks(ctx context.Context, source *store.Source, opts store.KW) error {
	includeExecuting := opts.Bool("include_executing")

	kw := store.KW{}
	if includeExecuting {
		kw = append(kw, store.Opt("include_executing", true))
	}

	if err := a.DeletePendingTasksFor(ctx, source, stringPtr("FastIndexingWorker"), kw); err != nil {
		return err
	}

	return a.DeletePendingTasksFor(ctx, source, stringPtr("MediaCollectionIndexingWorker"), kw)
}

// SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems/2
func (a *App) SlowIndexingHelpersIndexAndEnqueueDownloadForMediaItems(ctx context.Context, source *store.Source, opts store.KW) ([]any, error) {
	// source = Repo.preload(source, [:media_profile])
	source, err := a.PreloadSourceMediaProfile(ctx, source)
	if err != nil {
		return nil, err
	}

	// Setup file watcher and kickoff indexing
	mediaAttributes, err := slowIndexingHelpersSetupFileWatcherAndKickoffIndexing(ctx, a, source, opts)
	if err != nil {
		return nil, err
	}

	// store.Reload source because it may have been updated during indexing
	source, err = a.GetSource(ctx, source.ID)
	if err != nil {
		return nil, err
	}

	// Create media items
	result := make([]any, 0, len(mediaAttributes))
	for _, mediaAttrs := range mediaAttributes {
		mediaItem, err := a.CreateMediaItemFromBackendAttrs(ctx, source, mediaAttrs)
		if err != nil {
			// Return changeset errors
			if csErr, ok := err.(*store.ChangesetError); ok {
				result = append(result, csErr.Changeset)
			} else {
				result = append(result, err)
			}
		} else {
			result = append(result, mediaItem)
		}
	}

	// Update source's last_indexed_at
	if _, err := a.SourcesUpdateSource(ctx, source, store.Attrs{
		"last_indexed_at": db.Now(),
	}, store.KW{}); err != nil {
		return nil, err
	}

	// Enqueue pending download tasks
	if err := a.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, store.KW{}); err != nil {
		return nil, err
	}

	return result, nil
}

// Private helpers

func slowIndexingHelpersCalculateJobOffsetSeconds(source *store.Source) int {
	if source.LastIndexedAt == nil {
		return 0
	}

	offsetSeconds := int(time.Now().UTC().Sub(source.LastIndexedAt.Time).Seconds())
	indexFrequencySeconds := source.IndexFrequencyMinutes * 60

	if indexFrequencySeconds-offsetSeconds < 0 {
		return 0
	}
	return indexFrequencySeconds - offsetSeconds
}

func slowIndexingHelpersSetupFileWatcherAndKickoffIndexing(ctx context.Context, a *App, source *store.Source, opts store.KW) ([]*YtDlpMedia, error) {
	wasForced := opts.Bool("was_forced")

	// Start file follower
	fileFollower, err := FileFollowerServerStartLink(ctx, a.Config.FileWatcherPollInterval)
	if err != nil {
		return nil, err
	}

	// Setup handler
	handler := func(filepath string) {
		slowIndexingHelpersSetupFileFollowerWatcher(ctx, a, fileFollower, filepath, source)
	}

	shouldUseCookies := store.UseCookies(source, "indexing")

	// Build command options
	commandOpts := store.KW{
		store.Opt("output", a.DownloadOptionBuilderBuildOutputPathForSource(ctx, source)),
	}
	commandOpts = append(commandOpts, a.DownloadOptionBuilderBuildQualityOptionsForSource(ctx, source)...)
	commandOpts = append(commandOpts, slowIndexingHelpersBuildDownloadArchiveOptions(ctx, a, source, wasForced)...)

	// Build runner options
	runnerOpts := store.KW{
		store.Opt("file_listener_handler", handler),
		store.Opt("use_cookies", shouldUseCookies),
	}

	// Get media attributes
	result, err := a.MediaCollectionGetMediaAttributesForCollection(ctx, source.OriginalURL, commandOpts, runnerOpts)
	if err != nil {
		fileFollower.Stop()
		return nil, err
	}

	// Stop file follower
	fileFollower.Stop()

	return result, nil
}

func slowIndexingHelpersSetupFileFollowerWatcher(ctx context.Context, a *App, fileFollower *FileFollowerServer, filepath string, source *store.Source) {
	handler := func(line string) {
		// Decode JSON line
		var mediaAttrs map[string]any
		if err := json.Unmarshal([]byte(line), &mediaAttrs); err != nil {
			slog.Debug("FileFollowerServer Handler: Error decoding JSON", "error", err)
			return
		}

		slog.Debug("FileFollowerServer Handler: Got media attributes", "attrs", mediaAttrs)

		// Convert to YtDlpMedia struct (media_struct = YtDlpMedia.response_to_struct(media_attrs))
		mediaStruct := YtDlpMediaResponseToStruct(mediaAttrs)

		slowIndexingHelpersCreateMediaItemAndEnqueueDownload(ctx, a, source, mediaStruct)
	}

	fileFollower.WatchFile(filepath, handler)
}

func slowIndexingHelpersCreateMediaItemAndEnqueueDownload(ctx context.Context, a *App, source *store.Source, mediaAttrs *YtDlpMedia) {
	// store.Reload source in case it was updated during indexing
	reloadedSource, err := a.GetSource(ctx, source.ID)
	if err != nil {
		slog.Debug("FileFollowerServer Handler: Error reloading source", "error", err)
		return
	}

	// Create media item
	mediaItem, err := a.CreateMediaItemFromBackendAttrs(ctx, reloadedSource, mediaAttrs)
	if err != nil {
		slog.Debug("FileFollowerServer Handler: Error creating media item", "error", err)
		return
	}

	// Kickoff download if pending
	_, _ = a.DownloadingHelpersKickoffDownloadIfPending(ctx, mediaItem, store.KW{})
}

func slowIndexingHelpersBuildDownloadArchiveOptions(ctx context.Context, a *App, source *store.Source, wasForced bool) store.KW {
	// Don't use archive for playlists
	if source.CollectionType == store.SourceCollectionTypePlaylist {
		return store.KW{}
	}

	// Don't use archive if never indexed before
	if source.LastIndexedAt == nil {
		return store.KW{}
	}

	// Don't use archive if forced
	if wasForced {
		return store.KW{}
	}

	// Create archive file
	archiveFile, err := slowIndexingHelpersCreateDownloadArchiveFile(ctx, a, source)
	if err != nil {
		slog.Error("Error creating download archive file", "error", err)
		return store.KW{}
	}

	return store.KW{
		store.Flag("break_on_existing"),
		store.Opt("download_archive", archiveFile),
	}
}

func slowIndexingHelpersCreateDownloadArchiveFile(ctx context.Context, a *App, source *store.Source) (string, error) {
	tmpfile, err := fsutil.GenerateTmpfile(a.Config.TmpfileDirectory, "txt")
	if err != nil {
		return "", err
	}

	// Get media items for archive
	mediaItems, err := slowIndexingHelpersGetMediaItemsForDownloadArchive(ctx, a, source)
	if err != nil {
		return "", err
	}

	// Format archive contents (Enum.map_join("\n", ...): joined, no trailing newline)
	lines := make([]string, len(mediaItems))
	for i, item := range mediaItems {
		lines[i] = fmt.Sprintf("youtube %s", item.MediaID)
	}
	archiveContents := strings.Join(lines, "\n")

	// Write to file
	if err := os.WriteFile(tmpfile, []byte(archiveContents), 0644); err != nil {
		return "", err
	}

	return tmpfile, nil
}

func slowIndexingHelpersGetMediaItemsForDownloadArchive(ctx context.Context, a *App, source *store.Source) ([]*store.MediaItem, error) {
	q := store.MediaQueryNew().
		RequireAssoc("source").
		Where(store.MediaQueryForSource(source.ID)).
		Map(func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
			return b.OrderBy("uploaded_at DESC").Limit(50).Offset(20)
		})

	return store.All[store.MediaItem](ctx, a.Q(ctx), q)
}

func stringPtr(s string) *string {
	return &s
}
