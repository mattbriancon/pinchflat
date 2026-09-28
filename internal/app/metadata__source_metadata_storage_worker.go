package app

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

const SourceMetadataStorageWorkerName = "Pinchflat.Metadata.SourceMetadataStorageWorker"

var sourceMetadataStorageWorkerOpts = obanlite.WorkerOpts{
	Queue:       "remote_metadata",
	Tags:        []string{"media_source", "source_metadata", "remote_metadata", "show_in_dashboard"},
	MaxAttempts: 3,
}

// SourceMetadataStorageWorkerKickoffWithTask/2
func (a *App) SourceMetadataStorageWorkerKickoffWithTask(ctx context.Context, source *store.Source, opts store.KW) (*store.Task, error) {
	jobSpec := obanlite.JobSpec{
		Worker: SourceMetadataStorageWorkerName,
		Args:   map[string]any{"id": source.ID},
	}

	return a.CreateJobWithTask(ctx, jobSpec, source)
}

// perform/1
func (a *App) SourceMetadataStorageWorkerPerform(ctx context.Context, job *obanlite.Job) error {
	var args struct {
		ID int64 `json:"id"`
	}

	if err := job.DecodeArgs(&args); err != nil {
		return err
	}

	source, err := a.GetSource(ctx, args.ID)
	if err != nil {
		if err == store.ErrNotFound {
			slog.Info(fmt.Sprintf("%s discarded: source %d not found", SourceMetadataStorageWorkerName, args.ID))
			return nil
		}
		return err
	}

	// Preload associations
	source, err = a.PreloadSourceMetadata(ctx, source)
	if err != nil {
		if err == store.ErrNotFound {
			slog.Info(fmt.Sprintf("%s discarded: source %d stale", SourceMetadataStorageWorkerName, args.ID))
			return nil
		}
		return err
	}

	source, err = a.PreloadSourceMediaProfile(ctx, source)
	if err != nil {
		return err
	}

	seriesDirectory, err := determineSeriesDirectory(ctx, a, source)
	if err != nil {
		return err
	}

	sourceMetadata, sourceImageAttrs, metadataImageAttrs, err := fetchSourceMetadataAndImages(ctx, a, seriesDirectory, source)
	if err != nil {
		return err
	}

	sourceMetadataFilepath, err := a.MetadataFileHelpersCompressAndStoreMetadataFor(ctx, source, sourceMetadata)
	if err != nil {
		return err
	}

	var nfoFilepath any
	if source.MediaProfile != nil && source.MediaProfile.DownloadNfo && seriesDirectory != nil {
		nfoPath := filepath.Join(seriesDirectory.(string), "tvshow.nfo")
		_, err := NfoBuilderBuildAndStoreForSource(nfoPath, sourceMetadata)
		if err != nil {
			return err
		}
		nfoFilepath = nfoPath
	}

	// Merge update attributes
	updateAttrs := store.Attrs{
		"series_directory": seriesDirectory,
		"nfo_filepath":     nfoFilepath,
		"description":      sourceMetadata["description"],
		"metadata": map[string]any{
			"metadata_filepath": sourceMetadataFilepath,
		},
	}

	// Merge metadata image attributes
	for k, v := range metadataImageAttrs {
		if metadata, ok := updateAttrs["metadata"].(map[string]any); ok {
			metadata[k] = v
		}
	}

	// Merge source image attributes
	for k, v := range sourceImageAttrs {
		updateAttrs[k] = v
	}

	_, err = a.SourcesUpdateSource(ctx, source, updateAttrs, store.KW{store.Opt("run_post_commit_tasks", false)})
	return err
}

// determineSeriesDirectory/1
func determineSeriesDirectory(ctx context.Context, a *App, source *store.Source) (any, error) {
	outputPath := a.DownloadOptionBuilderBuildOutputPathForSource(ctx, source)

	runnerOpts := store.KW{store.Opt("output", outputPath)}
	addlOpts := store.KW{store.Opt("use_cookies", store.UseCookies(source, "metadata"))}

	sourceDetails, err := a.MediaCollectionGetSourceDetails(ctx, source.OriginalURL, runnerOpts, addlOpts)
	if err != nil {
		return nil, err
	}

	filepath, ok := sourceDetails["filepath"].(string)
	if !ok {
		return nil, fmt.Errorf("expected filepath in source details")
	}

	seriesDirectory, err := MetadataFileHelpersSeriesDirectoryFromMediaFilepath(filepath)
	if err != nil {
		return nil, nil // Return nil on error per Elixir {:error, _} -> nil
	}

	return seriesDirectory, nil
}

// fetchSourceMetadataAndImages/3
func fetchSourceMetadataAndImages(ctx context.Context, a *App, seriesDirectory any, source *store.Source) (map[string]any, store.Attrs, store.Attrs, error) {
	metadataDirectory, err := a.MetadataFileHelpersMetadataDirectoryFor(ctx, source)
	if err != nil {
		return nil, nil, nil, err
	}

	metadata, err := fetchMetadataForSource(ctx, a, source)
	if err != nil {
		return nil, nil, nil, err
	}

	metadataImageAttrsRaw, err := SourceImageParserStoreSourceImages(metadataDirectory, metadata)
	if err != nil {
		return nil, nil, nil, err
	}

	metadataImageAttrs := make(store.Attrs)
	for k, v := range metadataImageAttrsRaw {
		metadataImageAttrs[k] = v
	}

	sourceImageAttrs := store.Attrs{}
	if source.MediaProfile != nil && source.MediaProfile.DownloadSourceImages && seriesDirectory != nil {
		sourceImageAttrsRaw, err := SourceImageParserStoreSourceImages(seriesDirectory.(string), metadata)
		if err != nil {
			return nil, nil, nil, err
		}
		for k, v := range sourceImageAttrsRaw {
			sourceImageAttrs[k] = v
		}
	}

	return metadata, sourceImageAttrs, metadataImageAttrs, nil
}

// fetchMetadataForSource/1
func fetchMetadataForSource(ctx context.Context, a *App, source *store.Source) (map[string]any, error) {
	tmpDir := a.Config.TmpfileDirectory
	tmpOutputPath := filepath.Join(tmpDir, fsutil.RandomString(16), "source_image.%(ext)s")

	baseOpts := store.KW{
		store.Opt("convert_thumbnails", "jpg"),
		store.Opt("output", tmpOutputPath),
	}

	shouldUseCookies := store.UseCookies(source, "metadata")

	var opts store.KW
	if source.CollectionType == store.SourceCollectionTypeChannel {
		opts = append(baseOpts, store.Flag("write_all_thumbnails"), store.Opt("playlist_items", 0))
	} else {
		opts = append(baseOpts, store.Flag("write_thumbnail"), store.Opt("playlist_items", 1))
	}

	metadata, err := a.MediaCollectionGetSourceMetadata(ctx, source.OriginalURL, opts, store.KW{store.Opt("use_cookies", shouldUseCookies)})
	if err != nil {
		return nil, err
	}

	return metadata, nil
}
