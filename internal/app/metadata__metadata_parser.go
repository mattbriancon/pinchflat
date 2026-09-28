package app

import (
	"os"
	"sort"
)

// MetadataParserParseForMediaItem/1
func MetadataParserParseForMediaItem(metadata map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	// Merge results from all parsing functions
	for k, v := range parseMediaMetadata(metadata) {
		result[k] = v
	}
	for k, v := range parseSubtitleMetadata(metadata) {
		result[k] = v
	}
	for k, v := range parseThumbnailMetadata(metadata) {
		result[k] = v
	}
	for k, v := range parseInfojsonMetadata(metadata) {
		result[k] = v
	}

	return result, nil
}

// parseMediaMetadata extracts media-related fields from the metadata.
// Returns the struct fields from YtDlpMedia plus media_filepath.
func parseMediaMetadata(metadata map[string]any) map[string]any {
	ytMedia := YtDlpMediaResponseToStruct(metadata)

	result := make(map[string]any)
	result["media_id"] = ytMedia.MediaID
	result["title"] = ytMedia.Title
	result["description"] = ytMedia.Description
	result["original_url"] = ytMedia.OriginalURL
	result["livestream"] = ytMedia.Livestream
	result["short_form_content"] = ytMedia.ShortFormContent
	result["uploaded_at"] = ytMedia.UploadedAt
	result["duration_seconds"] = ytMedia.DurationSeconds
	result["predicted_media_filepath"] = ytMedia.PredictedMediaFilepath
	result["playlist_index"] = ytMedia.PlaylistIndex
	if fp, ok := metadata["filepath"].(string); ok {
		result["media_filepath"] = &fp
	} else {
		result["media_filepath"] = nil
	}

	return result
}

// parseSubtitleMetadata extracts subtitle filepaths from the metadata.
// Sorts them by language code.
func parseSubtitleMetadata(metadata map[string]any) map[string]any {
	subtitleFilepaths := make([][]string, 0)

	requestedSubs, ok := metadata["requested_subtitles"].(map[string]any)
	if ok {
		for lang, attrs := range requestedSubs {
			if attrsMap, ok := attrs.(map[string]any); ok {
				if filepath, ok := attrsMap["filepath"].(string); ok {
					subtitleFilepaths = append(subtitleFilepaths, []string{lang, filepath})
				}
			}
		}
	}

	// Sort by language code
	sort.Slice(subtitleFilepaths, func(i, j int) bool {
		return subtitleFilepaths[i][0] < subtitleFilepaths[j][0]
	})

	return map[string]any{
		"subtitle_filepaths": subtitleFilepaths,
	}
}

// parseThumbnailMetadata extracts and processes the thumbnail filepath.
// Reverses thumbnails list, finds the first with a filepath, and applies
// a workaround for a yt-dlp bug by inserting "-thumb" before the file extension.
func parseThumbnailMetadata(metadata map[string]any) map[string]any {
	var thumbnailFilepath *string

	thumbnails, ok := metadata["thumbnails"].([]any)
	if ok {
		// Reverse the thumbnails list
		for i := len(thumbnails) - 1; i >= 0; i-- {
			if thumbMap, ok := thumbnails[i].(map[string]any); ok {
				if fp, ok := thumbMap["filepath"].(string); ok {
					thumbnailFilepath = &fp
					break
				}
			}
		}
	}

	result := make(map[string]any)

	if thumbnailFilepath != nil {
		// Apply yt-dlp bug workaround: insert "-thumb" before the last 2 components
		// e.g., "file.jpg" -> "file-thumb.jpg", "path/file.webp" -> "path/file-thumb.webp"
		modified := applyThumbnailWorkaround(*thumbnailFilepath)
		result["thumbnail_filepath"] = filepathIfExists(modified)
	} else {
		result["thumbnail_filepath"] = nil
	}

	return result
}

// parseInfojsonMetadata extracts the infojson metadata filepath.
func parseInfojsonMetadata(metadata map[string]any) map[string]any {
	infojsonFilename, ok := metadata["infojson_filename"].(string)
	if !ok {
		infojsonFilename = ""
	}

	return map[string]any{
		"metadata_filepath": filepathIfExists(infojsonFilename),
	}
}

// applyThumbnailWorkaround applies the yt-dlp bug workaround by inserting "-thumb"
// before the file extension. This mimics the Elixir code's behavior of splitting
// on dots (with captures), inserting "-thumb" at position -3, and joining back.
// For example: "file.jpg" -> "file-thumb.jpg", "path/file.webp" -> "path/file-thumb.webp"
func applyThumbnailWorkaround(filepath string) string {
	// Find the last two dot positions
	lastDotIdx := -1
	secondLastDotIdx := -1

	for i := len(filepath) - 1; i >= 0; i-- {
		if filepath[i] == '.' {
			if lastDotIdx == -1 {
				lastDotIdx = i
			} else if secondLastDotIdx == -1 {
				secondLastDotIdx = i
				break
			}
		}
	}

	// If there's no dot, just append -thumb
	if lastDotIdx == -1 {
		return filepath + "-thumb"
	}

	// If there's only one dot, insert -thumb before it
	if secondLastDotIdx == -1 {
		return filepath[:lastDotIdx] + "-thumb" + filepath[lastDotIdx:]
	}

	// If there are multiple dots, insert before the second-to-last dot
	return filepath[:secondLastDotIdx] + "-thumb" + filepath[secondLastDotIdx:]
}

// filepathIfExists checks if a filepath exists on disk.
// Returns the filepath if it exists, nil otherwise.
// This is a workaround for a yt-dlp bug.
//
// Returns `any` (rather than *string) so that the "doesn't exist" case
// stores an untyped nil into the result map, not a typed-nil *string
// (which would compare != nil when read back out of a map[string]any).
func filepathIfExists(filepath string) any {
	if filepath == "" {
		return nil
	}

	if _, err := os.Stat(filepath); err == nil {
		return &filepath
	}

	return nil
}
