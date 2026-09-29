package app

import (
	"os"
	"sort"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// MetadataParserParseForMediaItem/1
func MetadataParserParseForMediaItem(metadata map[string]any) (store.MediaItemParams, error) {
	ytMedia := YtDlpMediaResponseToStruct(metadata)

	p := store.MediaItemParams{
		MediaID:                &ytMedia.MediaID,
		Title:                  &ytMedia.Title,
		Description:            &ytMedia.Description,
		OriginalURL:            &ytMedia.OriginalURL,
		Livestream:             &ytMedia.Livestream,
		ShortFormContent:       ytMedia.ShortFormContent,
		UploadedAt:             ytMedia.UploadedAt,
		DurationSeconds:        ytMedia.DurationSeconds,
		PredictedMediaFilepath: &ytMedia.PredictedMediaFilepath,
		PlaylistIndex:          ytMedia.PlaylistIndex,
	}
	if ytMedia.ShortFormContent == nil {
		p.Clear |= store.ClearShortFormContent
	}
	if ytMedia.UploadedAt == nil {
		p.Clear |= store.ClearUploadedAt
	}

	if fp, ok := metadata["filepath"].(string); ok {
		p.MediaFilepath = &fp
	} else {
		p.Clear |= store.ClearMediaFilepath
	}

	subtitles := parseSubtitleMetadata(metadata)
	p.SubtitleFilepaths = &subtitles

	if thumbnail := parseThumbnailMetadata(metadata); thumbnail != nil {
		p.ThumbnailFilepath = thumbnail
	} else {
		p.Clear |= store.ClearThumbnailFilepath
	}

	infojsonFilename, _ := metadata["infojson_filename"].(string)
	if infojson := filepathIfExists(infojsonFilename); infojson != nil {
		p.MetadataFilepath = infojson
	} else {
		p.Clear |= store.ClearMetadataFilepath
	}

	return p, nil
}

// parseSubtitleMetadata extracts subtitle filepaths from the metadata.
// Sorts them by language code.
func parseSubtitleMetadata(metadata map[string]any) db.NestedStringArray {
	subtitleFilepaths := db.NestedStringArray{}

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

	return subtitleFilepaths
}

// parseThumbnailMetadata extracts and processes the thumbnail filepath.
// Reverses thumbnails list, finds the first with a filepath, and applies
// a workaround for a yt-dlp bug by inserting "-thumb" before the file extension.
// Returns nil if there is no thumbnail on disk.
func parseThumbnailMetadata(metadata map[string]any) *string {
	thumbnails, ok := metadata["thumbnails"].([]any)
	if !ok {
		return nil
	}

	// Reverse the thumbnails list
	for i := len(thumbnails) - 1; i >= 0; i-- {
		if thumbMap, ok := thumbnails[i].(map[string]any); ok {
			if fp, ok := thumbMap["filepath"].(string); ok {
				// Apply yt-dlp bug workaround: insert "-thumb" before the last 2 components
				// e.g., "file.jpg" -> "file-thumb.jpg", "path/file.webp" -> "path/file-thumb.webp"
				return filepathIfExists(applyThumbnailWorkaround(fp))
			}
		}
	}
	return nil
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
func filepathIfExists(filepath string) *string {
	if filepath == "" {
		return nil
	}

	if _, err := os.Stat(filepath); err == nil {
		return &filepath
	}

	return nil
}
