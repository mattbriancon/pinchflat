package web

// Port of lib/pinchflat_web/controllers/media_items/media_item_html.ex

import (
	"path/filepath"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// mediaFileExists checks if a media file exists on disk.
func mediaFileExists(mediaItem *core.MediaItem) bool {
	if mediaItem == nil || mediaItem.MediaFilepath == nil || *mediaItem.MediaFilepath == "" {
		return false
	}
	return fileExists(*mediaItem.MediaFilepath)
}

// mediaType returns the type of media based on file extension.
// Returns "video", "audio", or "unknown".
func mediaType(mediaItem *core.MediaItem) string {
	if mediaItem == nil || mediaItem.MediaFilepath == nil || *mediaItem.MediaFilepath == "" {
		return "unknown"
	}
	ext := strings.ToLower(filepath.Ext(*mediaItem.MediaFilepath))
	switch ext {
	case ".mp4", ".webm", ".mkv":
		return "video"
	case ".mp3", ".m4a", ".opus":
		return "audio"
	default:
		return "unknown"
	}
}
