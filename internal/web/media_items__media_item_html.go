package web

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

// fieldRef takes the address of a Form.Field result so it can be passed as
// CoreInputProps.Field (a form component call in Elixir passes the
// FormField by value; Go's CoreInputProps wants a pointer).
func fieldRef(f FormField) *FormField { return &f }

func miStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
