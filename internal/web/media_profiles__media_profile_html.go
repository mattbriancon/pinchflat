package web

// Port of lib/pinchflat_web/controllers/media_profiles/media_profile_html.ex

import (
	"fmt"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// formatInt64 formats an int64 as a string.
func formatInt64(v int64) string {
	return fmt.Sprint(v)
}

// formatBool formats a bool as a string.
func formatBool(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

// convertOptions converts interface{} slice to CoreSelectOption slice.
func convertOptions(options []interface{}) []CoreSelectOption {
	out := make([]CoreSelectOption, 0, len(options))
	for _, opt := range options {
		if m, ok := opt.(map[string]interface{}); ok {
			label := fmt.Sprint(m["label"])
			value := fmt.Sprint(m["value"])
			out = append(out, CoreSelectOption{Label: label, Value: value})
		}
	}
	return out
}

// Helper functions for media profile templates.

func friendlyFormatTypeOptions() []interface{} {
	return []interface{}{
		map[string]interface{}{"label": "Include (default)", "value": "include"},
		map[string]interface{}{"label": "Exclude", "value": "exclude"},
		map[string]interface{}{"label": "Only", "value": "only"},
	}
}

func friendlyResolutionOptions() []interface{} {
	return []interface{}{
		map[string]interface{}{"label": "8k", "value": "4320p"},
		map[string]interface{}{"label": "4k", "value": "2160p"},
		map[string]interface{}{"label": "1440p", "value": "1440p"},
		map[string]interface{}{"label": "1080p", "value": "1080p"},
		map[string]interface{}{"label": "720p", "value": "720p"},
		map[string]interface{}{"label": "480p", "value": "480p"},
		map[string]interface{}{"label": "360p", "value": "360p"},
		map[string]interface{}{"label": "Audio Only", "value": "audio"},
	}
}

func friendlySponsorblockOptions() []interface{} {
	return []interface{}{
		map[string]interface{}{"label": "Disabled (default)", "value": "disabled"},
		map[string]interface{}{"label": "Mark Segments as Chapters", "value": "mark"},
		map[string]interface{}{"label": "Remove Segments", "value": "remove"},
	}
}

func friendlySponsorblockCategories() []interface{} {
	return []interface{}{
		map[string]interface{}{"label": "Sponsor", "value": "sponsor"},
		map[string]interface{}{"label": "Intro/Intermission", "value": "intro"},
		map[string]interface{}{"label": "Outro/Credits", "value": "outro"},
		map[string]interface{}{"label": "Self Promotion", "value": "selfpromo"},
		map[string]interface{}{"label": "Preview/Recap", "value": "preview"},
		map[string]interface{}{"label": "Filler Tangent", "value": "filler"},
		map[string]interface{}{"label": "Interaction Reminder", "value": "interaction"},
		map[string]interface{}{"label": "Non-music Section", "value": "music_offtopic"},
	}
}

func mediaCenterCustomOutputTemplateOptions() []interface{} {
	return []interface{}{
		map[string]interface{}{
			"key": "season_by_year__episode_by_date",
			"val": "<code>Season YYYY/sYYYYeMMDD</code>",
		},
		map[string]interface{}{
			"key": "season_by_year__episode_by_date_and_index",
			"val": "same as the above but it handles dates better. <strong>This is the recommended option</strong>",
		},
		map[string]interface{}{
			"key": "static_season__episode_by_index",
			"val": "<code>Season 1/s01eXX</code> where <code>XX</code> is the video's position in the playlist. Only recommended for playlists (not channels) that don't change",
		},
		map[string]interface{}{
			"key": "static_season__episode_by_date",
			"val": "<code>Season 1/s01eYYMMDD</code>. Recommended for playlists that might change or where order isn't important",
		},
	}
}

func otherCustomOutputTemplateOptions() []interface{} {
	return []interface{}{
		map[string]interface{}{"key": "upload_day", "val": ""},
		map[string]interface{}{"key": "upload_month", "val": ""},
		map[string]interface{}{"key": "upload_year", "val": ""},
		map[string]interface{}{"key": "upload_yyyy_mm_dd", "val": "the upload date in the format <code>YYYY-MM-DD</code>"},
		map[string]interface{}{"key": "source_custom_name", "val": "the name of the sources that use this profile"},
		map[string]interface{}{"key": "source_collection_id", "val": "the YouTube ID of the sources that use this profile"},
		map[string]interface{}{"key": "source_collection_name", "val": "the YouTube name of the sources that use this profile (often the same as source_custom_name)"},
		map[string]interface{}{"key": "source_collection_type", "val": "the collection type of the sources using this profile. Either 'channel' or 'playlist'"},
		map[string]interface{}{"key": "artist_name", "val": "the name of the artist with fallbacks to other uploader fields"},
		map[string]interface{}{"key": "season_from_date", "val": "alias for upload_year"},
		map[string]interface{}{"key": "season_episode_from_date", "val": "the upload date formatted as <code>sYYYYeMMDD</code>"},
		map[string]interface{}{"key": "season_episode_index_from_date", "val": "the upload date formatted as <code>sYYYYeMMDDII</code> where <code>II</code> is an index to prevent date collisions"},
		map[string]interface{}{"key": "media_playlist_index", "val": "the place of the media item in the playlist. Do not use with channels. May not work if the playlist is updated"},
		map[string]interface{}{"key": "media_item_id", "val": "the ID of the media item in Pinchflat's database"},
		map[string]interface{}{"key": "source_id", "val": "the ID of the source in Pinchflat's database"},
		map[string]interface{}{"key": "media_profile_id", "val": "the ID of the media profile in Pinchflat's database"},
	}
}

func commonOutputTemplateOptions() []string {
	return []string{"id", "ext", "title", "uploader", "channel", "upload_date", "duration_string"}
}

func presetOptions() []interface{} {
	return []interface{}{
		map[string]interface{}{"label": "Default", "value": "default"},
		map[string]interface{}{"label": "Media Center (Plex, Jellyfin, Kodi, etc.)", "value": "media_center"},
		map[string]interface{}{"label": "Music", "value": "audio"},
		map[string]interface{}{"label": "Archiving", "value": "archiving"},
	}
}

func defaultOutputTemplate() string {
	return (&core.MediaProfile{}).OutputPathTemplate
}

func mediaCenterOutputTemplate() string {
	return "/shows/{{ source_custom_name }}/{{ season_by_year__episode_by_date_and_index }} - {{ title }}.{{ ext }}"
}

func audioOutputTemplate() string {
	return "/music/{{ artist_name }}/{{ title }}.{{ ext }}"
}

func archivingOutputTemplate() string {
	return defaultOutputTemplate()
}
