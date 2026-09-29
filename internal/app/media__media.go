package app

import (
	"context"
	"fmt"
	"os"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// DeleteMediaItem/1 and DeleteMediaItem/2
//
// Deletes a media_item, its associated tasks, and our internal metadata
// files. Can optionally delete the media_item's media files (media,
// thumbnail, subtitles, etc).
func (a *App) MediaDeleteMediaItem(ctx context.Context, mediaItem *store.MediaItem, deleteFiles bool) (*store.MediaItem, error) {
	if err := a.DeleteTasksFor(ctx, mediaItem, nil, obanlite.AllStates); err != nil {
		return nil, err
	}

	if deleteFiles {
		if err := mediaDoDeleteMediaFiles(ctx, mediaItem); err != nil {
			return nil, err
		}
		// The Elixir code doesn't check the result of this call.
		_ = a.UserScripts.Run(ctx, "media_deleted", mediaItem)
	}

	// Should delete these no matter what
	// The Elixir code doesn't check the result of this call.
	_ = a.mediaDeleteInternalMetadataFiles(ctx, mediaItem)

	if err := store.Delete[store.MediaItem](ctx, a.Q(ctx), mediaItem); err != nil {
		return nil, err
	}
	return mediaItem, nil
}

// DeleteMediaFiles/1 and DeleteMediaFiles/2
//
// Deletes the tasks and media files associated with a media_item but leaves
// the media_item in the database. Does not delete anything to do with
// associated metadata.
//
// extra is merged into the media_item before it is updated. Useful for
// setting things like `prevent_download` and `culled_at`, if wanted.
func (a *App) MediaDeleteMediaFiles(ctx context.Context, mediaItem *store.MediaItem, extra store.MediaItemParams) (*store.MediaItem, error) {
	if err := a.DeleteTasksFor(ctx, mediaItem, nil, obanlite.AllStates); err != nil {
		return nil, err
	}
	if err := mediaDoDeleteMediaFiles(ctx, mediaItem); err != nil {
		return nil, err
	}
	// The Elixir code doesn't check the result of this call.
	_ = a.UserScripts.Run(ctx, "media_deleted", mediaItem)

	extra.Clear |= store.ClearFilepaths
	extra.SubtitleFilepaths = &db.NestedStringArray{}
	return a.UpdateMediaItem(ctx, mediaItem, extra)
}

// do_delete_media_files/1
func mediaDoDeleteMediaFiles(ctx context.Context, mediaItem *store.MediaItem) error {
	var paths []string
	for _, field := range store.MediaItemFilepathAttributes() {
		switch field {
		case "subtitle_filepaths":
			for _, pair := range mediaItem.SubtitleFilepaths {
				if len(pair) >= 2 {
					paths = append(paths, pair[1])
				}
			}
		case "media_filepath":
			if mediaItem.MediaFilepath != nil {
				paths = append(paths, *mediaItem.MediaFilepath)
			}
		case "thumbnail_filepath":
			if mediaItem.ThumbnailFilepath != nil {
				paths = append(paths, *mediaItem.ThumbnailFilepath)
			}
		case "metadata_filepath":
			if mediaItem.MetadataFilepath != nil {
				paths = append(paths, *mediaItem.MetadataFilepath)
			}
		case "nfo_filepath":
			if mediaItem.NfoFilepath != nil {
				paths = append(paths, *mediaItem.NfoFilepath)
			}
		}
	}

	// Mirrors Elixir's Enum.each/2, which discards each call's return value
	// (including "file not found" errors) rather than aborting the deletion.
	for _, p := range paths {
		_ = fsutil.DeleteFileAndRemoveEmptyDirs(p)
	}
	return nil
}

// delete_internal_metadata_files/1
func (a *App) mediaDeleteInternalMetadataFiles(ctx context.Context, mediaItem *store.MediaItem) error {
	if _, err := a.PreloadMediaItemMetadata(ctx, mediaItem); err != nil {
		return err
	}

	metadata := mediaItem.Metadata
	if metadata == nil {
		metadata = store.NewMediaMetadata()
	}

	for _, field := range store.MediaMetadataFilepathAttributes() {
		var path string
		switch field {
		case "metadata_filepath":
			path = metadata.MetadataFilepath
		case "thumbnail_filepath":
			path = metadata.ThumbnailFilepath
		}
		if path == "" {
			continue
		}
		// Mirrors Elixir's Enum.each/2, which discards each call's return
		// value rather than aborting on a "file not found" error.
		_ = fsutil.DeleteFileAndRemoveEmptyDirs(path)
	}
	return nil
}

// ComputeAndSaveMediaFilesize fetches the on-disk size of a media item's
// file and saves it to the database.
func (a *App) ComputeAndSaveMediaFilesize(ctx context.Context, mediaItem *store.MediaItem) (*store.MediaItem, error) {
	if mediaItem.MediaFilepath == nil {
		return nil, fmt.Errorf("media_filepath is nil")
	}

	stat, err := os.Stat(*mediaItem.MediaFilepath)
	if err != nil {
		return nil, err
	}

	return a.UpdateMediaItem(ctx, mediaItem, store.MediaItemParams{MediaSizeBytes: store.Ptr(stat.Size())})
}
