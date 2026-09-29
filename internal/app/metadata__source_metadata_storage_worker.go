package app

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"path/filepath"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

const SourceMetadataStorageWorkerName = "Pinchflat.Metadata.SourceMetadataStorageWorker"

var sourceMetadataStorageWorkerOpts = obanlite.WorkerOpts{
	Queue:       "remote_metadata",
	Tags:        []string{"media_source", "source_metadata", "remote_metadata", "show_in_dashboard"},
	MaxAttempts: 3,
}

// SourceMetadataStorageWorkerKickoffWithTask/2
func (a *App) SourceMetadataStorageWorkerKickoffWithTask(ctx context.Context, source *store.Source) (*store.Task, error) {
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

	sourceMetadata, sourceImages, metadataImages, err := fetchSourceMetadataAndImages(ctx, a, seriesDirectory, source)
	if err != nil {
		return err
	}

	sourceMetadataFilepath, err := a.MetadataFileHelpersCompressAndStoreMetadataFor(ctx, source, sourceMetadata)
	if err != nil {
		return err
	}

	// A missing series directory, nfo file or description is submitted blank,
	// which clears the column.
	seriesDir, _ := seriesDirectory.(string)
	nfoFilepath := ""
	if source.MediaProfile != nil && source.MediaProfile.DownloadNfo && seriesDirectory != nil {
		nfoPath := filepath.Join(seriesDir, "tvshow.nfo")
		_, err := NfoBuilderBuildAndStoreForSource(nfoPath, sourceMetadata)
		if err != nil {
			return err
		}
		nfoFilepath = nfoPath
	}
	description, _ := sourceMetadata["description"].(string)

	p := store.SourceParams{
		SeriesDirectory: &seriesDir,
		NfoFilepath:     &nfoFilepath,
		Description:     &description,
		Metadata: &store.SourceMetadata{
			MetadataFilepath: sourceMetadataFilepath,
			FanartFilepath:   imagePath(metadataImages, "fanart_filepath"),
			PosterFilepath:   imagePath(metadataImages, "poster_filepath"),
			BannerFilepath:   imagePath(metadataImages, "banner_filepath"),
		},
		FanartFilepath: imagePath(sourceImages, "fanart_filepath"),
		PosterFilepath: imagePath(sourceImages, "poster_filepath"),
		BannerFilepath: imagePath(sourceImages, "banner_filepath"),
	}
	_, err = a.SourcesUpdateSource(ctx, source, p, false)
	return err
}

// imagePath is the stored path for an image attribute, or nil when there is
// none (which leaves the column as it is).
func imagePath(images map[string]string, attr string) *string {
	if path, ok := images[attr]; ok && path != "" {
		return &path
	}
	return nil
}

// determineSeriesDirectory/1
func determineSeriesDirectory(ctx context.Context, a *App, source *store.Source) (any, error) {
	outputPath := a.DownloadOptionBuilderBuildOutputPathForSource(ctx, source)

	args := ytdlp.Args{}.Opt("output", outputPath)
	callOpts := ytdlp.CallOptions{UseCookies: store.UseCookies(source, "metadata")}

	sourceDetails, err := a.MediaCollectionGetSourceDetails(ctx, source.OriginalURL, args, callOpts)
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

// fetchSourceMetadataAndImages/3 returns the source's metadata, and the
// image paths stored for the source and for its metadata.
func fetchSourceMetadataAndImages(ctx context.Context, a *App, seriesDirectory any, source *store.Source) (map[string]any, map[string]string, map[string]string, error) {
	metadataDirectory, err := a.MetadataFileHelpersMetadataDirectoryFor(ctx, source)
	if err != nil {
		return nil, nil, nil, err
	}

	metadata, err := fetchMetadataForSource(ctx, a, source)
	if err != nil {
		return nil, nil, nil, err
	}

	metadataImages, err := SourceImageParserStoreSourceImages(metadataDirectory, metadata)
	if err != nil {
		return nil, nil, nil, err
	}

	var sourceImages map[string]string
	if source.MediaProfile != nil && source.MediaProfile.DownloadSourceImages && seriesDirectory != nil {
		sourceImages, err = SourceImageParserStoreSourceImages(seriesDirectory.(string), metadata)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return metadata, sourceImages, metadataImages, nil
}

// fetchMetadataForSource/1
func fetchMetadataForSource(ctx context.Context, a *App, source *store.Source) (map[string]any, error) {
	tmpDir := a.Config.TmpfileDirectory
	tmpOutputPath := filepath.Join(tmpDir, fmt.Sprintf("%016x", rand.Uint64()), "source_image.%(ext)s")

	args := ytdlp.Args{}.Opt("convert_thumbnails", "jpg").Opt("output", tmpOutputPath)
	if source.CollectionType == store.SourceCollectionTypeChannel {
		args = args.Flag("write_all_thumbnails").Opt("playlist_items", 0)
	} else {
		args = args.Flag("write_thumbnail").Opt("playlist_items", 1)
	}

	callOpts := ytdlp.CallOptions{UseCookies: store.UseCookies(source, "metadata")}
	metadata, err := a.MediaCollectionGetSourceMetadata(ctx, source.OriginalURL, args, callOpts)
	if err != nil {
		return nil, err
	}

	return metadata, nil
}
