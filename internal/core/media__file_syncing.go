package core

import (
	"context"
	"os"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

// DeleteOutdatedFiles/2
func FileSyncingDeleteOutdatedFiles(ctx context.Context, oldMediaItem, newMediaItem *MediaItem) error {
	nonSubtitleKeys := fileSyncingNonSubtitleFilepathAttributes()

	oldNonSubtitles := fileSyncingExtractFilepaths(oldMediaItem, nonSubtitleKeys)
	oldSubtitles := fileSyncingFromNestedList(oldMediaItem.SubtitleFilepaths)
	newNonSubtitles := fileSyncingExtractFilepaths(newMediaItem, nonSubtitleKeys)
	newSubtitles := fileSyncingFromNestedList(newMediaItem.SubtitleFilepaths)

	fileSyncingHandleFileDeletion(ctx, oldNonSubtitles, newNonSubtitles)
	fileSyncingHandleFileDeletion(ctx, oldSubtitles, newSubtitles)

	return nil
}

// SyncFilePresenceOnDisk/1
func (a *App) FileSyncingSyncFilePresenceOnDisk(ctx context.Context, mediaItems []*MediaItem) ([]*MediaItem, error) {
	result := make([]*MediaItem, len(mediaItems))

	for i, mediaItem := range mediaItems {
		newAttributes := fileSyncingSyncMediaItemFiles(mediaItem)
		// Doing this one-by-one instead of batching since this process
		// can take time and a batch could let MediaItem state get out of sync
		updatedMediaItem, err := a.MediaUpdateMediaItem(ctx, mediaItem, newAttributes)
		if err != nil {
			return nil, err
		}

		result[i] = updatedMediaItem
	}

	return result, nil
}

// Private helpers

func fileSyncingNonSubtitleFilepathAttributes() []string {
	attrs := []string{
		"media_filepath",
		"thumbnail_filepath",
		"metadata_filepath",
		"nfo_filepath",
	}
	return attrs
}

func fileSyncingExtractFilepaths(mediaItem *MediaItem, keys []string) map[string]*string {
	result := make(map[string]*string)

	for _, key := range keys {
		switch key {
		case "media_filepath":
			result[key] = mediaItem.MediaFilepath
		case "thumbnail_filepath":
			result[key] = mediaItem.ThumbnailFilepath
		case "metadata_filepath":
			result[key] = mediaItem.MetadataFilepath
		case "nfo_filepath":
			result[key] = mediaItem.NfoFilepath
		}
	}

	return result
}

func fileSyncingFromNestedList(nestedList [][]string) map[string]*string {
	result := make(map[string]*string)

	for _, pair := range nestedList {
		if len(pair) == 2 {
			result[pair[0]] = &pair[1]
		}
	}

	return result
}

func fileSyncingHandleFileDeletion(ctx context.Context, oldAttributes, newAttributes map[string]*string) {
	// The logic:
	//   - A file should only be deleted if it exists and the new file is different
	//   - The new attributes are the ones we're interested in keeping
	//   - If the old attributes have a key that doesn't exist in the new attributes, don't touch it.
	//     This is good for archiving but may be unpopular for other users so this may change.

	for key, newFilepath := range newAttributes {
		oldFilepath := oldAttributes[key]
		filesHaveChanged := oldFilepath != nil && newFilepath != nil && *oldFilepath != *newFilepath
		filesExistOnDisk := filesHaveChanged && fileExists(*oldFilepath) && fileExists(*newFilepath)

		if filesExistOnDisk && !fsutil.SameFile(*oldFilepath, *newFilepath) {
			fsutil.DeleteFileAndRemoveEmptyDirs(*oldFilepath)
		}
	}
}

func fileSyncingSyncMediaItemFiles(mediaItem *MediaItem) Attrs {
	nonSubtitleKeys := fileSyncingNonSubtitleFilepathAttributes()
	subtitleKeys := fileSyncingFromNestedList(mediaItem.SubtitleFilepaths)
	nonSubtitles := fileSyncingExtractFilepaths(mediaItem, nonSubtitleKeys)

	// This one is checking for the negative (ie: only update if the file doesn't exist)
	newNonSubtitleAttrs := make(Attrs)
	for key, filepath := range nonSubtitles {
		if filepath == nil || !fileExists(*filepath) {
			newNonSubtitleAttrs[key] = nil
		}
	}

	// This one is checking for the positive (ie: only update if the file exists)
	// This is because subtitles, being an array type in the DB, are most easily updated
	// by a full replacement rather than finding the actual diff
	var newSubtitleAttrs [][]string
	for key, filepath := range subtitleKeys {
		if filepath != nil && fileExists(*filepath) {
			newSubtitleAttrs = append(newSubtitleAttrs, []string{key, *filepath})
		}
	}

	newNonSubtitleAttrs["subtitle_filepaths"] = newSubtitleAttrs

	return newNonSubtitleAttrs
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
