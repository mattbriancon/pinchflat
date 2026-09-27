package web

// Port of lib/pinchflat_web/controllers/media_items/media_item_html.ex

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db"
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

// mediaItemAttr is one row of core_components.ex's list_items_from_map/1,
// scoped to MediaItem's own schema fields (see the report: list_items_from_map
// itself doesn't exist in the ported core_components, so it's replicated
// here for this row only).
type mediaItemAttr struct {
	Key   string
	Value string
	IsURL bool
}

// miNestedStringArray renders subtitle_filepaths ([[lang, path], ...]) the
// way list_items_from_map/1 renders any list value: Enum.join(v, ", ").
func miNestedStringArray(v db.NestedStringArray) string {
	parts := make([]string, len(v))
	for i, inner := range v {
		parts[i] = strings.Join(inner, ":")
	}
	return strings.Join(parts, ", ")
}

func miIsHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func miStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func miInt64(p *int64) string {
	if p == nil {
		return ""
	}
	return strconv.FormatInt(*p, 10)
}

func miInt(p *int) string {
	if p == nil {
		return ""
	}
	return strconv.Itoa(*p)
}

func miBool(p *bool) string {
	if p == nil {
		return ""
	}
	return fmt.Sprint(*p)
}

// miUTCDateTimeStr renders a nullable db.UTCDateTime the way Elixir's
// %DateTime{} struct is shown by list_items_from_map/1 (via to_string/1,
// i.e. ISO 8601).
func miUTCDateTimeStr(t *db.UTCDateTime) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02T15:04:05Z")
}

// mediaItemRawAttributes builds the rows list_items_from_map/1 would render
// for `Map.from_struct(@media_item)`: every real column, in schema order
// (associations and the virtual matching_search_term are always dropped by
// the Elixir filter, so they're left out here too).
func mediaItemRawAttributes(m *core.MediaItem) []mediaItemAttr {
	if m == nil {
		return nil
	}
	rows := []mediaItemAttr{
		{"id", strconv.FormatInt(m.ID, 10), false},
		{"media_id", m.MediaID, false},
		{"title", miStr(m.Title), false},
		{"media_filepath", miStr(m.MediaFilepath), false},
		{"source_id", strconv.FormatInt(m.SourceID, 10), false},
		{"subtitle_filepaths", miNestedStringArray(m.SubtitleFilepaths), false},
		{"thumbnail_filepath", miStr(m.ThumbnailFilepath), false},
		{"metadata_filepath", miStr(m.MetadataFilepath), false},
		{"livestream", fmt.Sprint(m.Livestream), false},
		{"original_url", m.OriginalURL, miIsHTTPURL(m.OriginalURL)},
		{"media_downloaded_at", miUTCDateTimeStr(m.MediaDownloadedAt), false},
		{"details_updated_at", miUTCDateTimeStr(m.DetailsUpdatedAt), false},
		{"description", miStr(m.Description), false},
		{"media_size_bytes", miInt64(m.MediaSizeBytes), false},
		{"short_form_content", fmt.Sprint(m.ShortFormContent), false},
		{"uploaded_at", m.UploadedAt.Time.Format("2006-01-02T15:04:05Z"), false},
		{"nfo_filepath", miStr(m.NfoFilepath), false},
		{"uuid", miStr(m.UUID), false},
		{"duration_seconds", miInt(m.DurationSeconds), false},
		{"prevent_download", fmt.Sprint(m.PreventDownload), false},
		{"culled_at", miUTCDateTimeStr(m.CulledAt), false},
		{"prevent_culling", miBool(m.PreventCulling), false},
		{"media_redownloaded_at", miUTCDateTimeStr(m.MediaRedownloadedAt), false},
		{"upload_date_index", strconv.Itoa(m.UploadDateIndex), false},
		{"playlist_index", strconv.Itoa(m.PlaylistIndex), false},
		{"predicted_media_filepath", miStr(m.PredictedMediaFilepath), false},
		{"last_error", miStr(m.LastError), false},
		{"inserted_at", m.InsertedAt.Time.Format("2006-01-02T15:04:05Z"), false},
		{"updated_at", m.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"), false},
	}
	return rows
}
