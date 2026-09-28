package web

// The renderable templates live in the sibling .templ files. Helper names
// use the mp prefix to avoid collisions with other controllers' files in
// this flat package.

import (
	"github.com/mattbriancon/pinchflat/internal/fsutil"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// mpKV is one entry of media_center_custom_output_template_options/0 and
// other_custom_output_template_options/0 (an alias key plus optional help,
// already-safe HTML like the Elixir source).
type mpKV struct {
	Key   string
	Value string
}

// mediaProfilesFriendlyFormatTypeOptions is friendly_format_type_options/0.
func mediaProfilesFriendlyFormatTypeOptions() []CoreSelectOption {
	return []CoreSelectOption{
		{Label: "Include (default)", Value: "include"},
		{Label: "Exclude", Value: "exclude"},
		{Label: "Only", Value: "only"},
	}
}

// mediaProfilesFriendlyResolutionOptions is friendly_resolution_options/0.
func mediaProfilesFriendlyResolutionOptions() []CoreSelectOption {
	return []CoreSelectOption{
		{Label: "8k", Value: "4320p"},
		{Label: "4k", Value: "2160p"},
		{Label: "1440p", Value: "1440p"},
		{Label: "1080p", Value: "1080p"},
		{Label: "720p", Value: "720p"},
		{Label: "480p", Value: "480p"},
		{Label: "360p", Value: "360p"},
		{Label: "Audio Only", Value: "audio"},
	}
}

// mediaProfilesFriendlySponsorblockOptions is friendly_sponsorblock_options/0.
func mediaProfilesFriendlySponsorblockOptions() []CoreSelectOption {
	return []CoreSelectOption{
		{Label: "Disabled (default)", Value: "disabled"},
		{Label: "Mark Segments as Chapters", Value: "mark"},
		{Label: "Remove Segments", Value: "remove"},
	}
}

// mediaProfilesFriendlySponsorblockCategories is frieldly_sponsorblock_categories/0
// (sic, matching the Elixir typo in the function name only, not the Go name).
func mediaProfilesFriendlySponsorblockCategories() []CoreSelectOption {
	return []CoreSelectOption{
		{Label: "Sponsor", Value: "sponsor"},
		{Label: "Intro/Intermission", Value: "intro"},
		{Label: "Outro/Credits", Value: "outro"},
		{Label: "Self Promotion", Value: "selfpromo"},
		{Label: "Preview/Recap", Value: "preview"},
		{Label: "Filler Tangent", Value: "filler"},
		{Label: "Interaction Reminder", Value: "interaction"},
		{Label: "Non-music Section", Value: "music_offtopic"},
	}
}

// mediaProfilesMediaCenterCustomOutputTemplateOptions is
// media_center_custom_output_template_options/0.
func mediaProfilesMediaCenterCustomOutputTemplateOptions() []mpKV {
	return []mpKV{
		{Key: "season_by_year__episode_by_date", Value: "<code>Season YYYY/sYYYYeMMDD</code>"},
		{
			Key: "season_by_year__episode_by_date_and_index",
			Value: "same as the above but it handles dates better. " +
				"<strong>This is the recommended option</strong>",
		},
		{
			Key: "static_season__episode_by_index",
			Value: "<code>Season 1/s01eXX</code> where <code>XX</code> is the video's position in the playlist. " +
				"Only recommended for playlists (not channels) that don't change",
		},
		{
			Key: "static_season__episode_by_date",
			Value: "<code>Season 1/s01eYYMMDD</code>. Recommended for playlists that might change or where " +
				"order isn't important",
		},
	}
}

// mediaProfilesOtherCustomOutputTemplateOptions is
// other_custom_output_template_options/0.
func mediaProfilesOtherCustomOutputTemplateOptions() []mpKV {
	return []mpKV{
		{Key: "upload_day"},
		{Key: "upload_month"},
		{Key: "upload_year"},
		{Key: "upload_yyyy_mm_dd", Value: "the upload date in the format <code>YYYY-MM-DD</code>"},
		{Key: "source_custom_name", Value: "the name of the sources that use this profile"},
		{Key: "source_collection_id", Value: "the YouTube ID of the sources that use this profile"},
		{
			Key: "source_collection_name",
			Value: "the YouTube name of the sources that use this profile (often the same as " +
				"source_custom_name)",
		},
		{
			Key:   "source_collection_type",
			Value: "the collection type of the sources using this profile. Either 'channel' or 'playlist'",
		},
		{Key: "artist_name", Value: "the name of the artist with fallbacks to other uploader fields"},
		{Key: "season_from_date", Value: "alias for upload_year"},
		{Key: "season_episode_from_date", Value: "the upload date formatted as <code>sYYYYeMMDD</code>"},
		{
			Key: "season_episode_index_from_date",
			Value: "the upload date formatted as <code>sYYYYeMMDDII</code> where <code>II</code> is an index " +
				"to prevent date collisions",
		},
		{
			Key: "media_playlist_index",
			Value: "the place of the media item in the playlist. Do not use with channels. May not work if " +
				"the playlist is updated",
		},
		{Key: "media_item_id", Value: "the ID of the media item in Pinchflat's database"},
		{Key: "source_id", Value: "the ID of the source in Pinchflat's database"},
		{Key: "media_profile_id", Value: "the ID of the media profile in Pinchflat's database"},
	}
}

// mediaProfilesCommonOutputTemplateOptions is common_output_template_options/0.
func mediaProfilesCommonOutputTemplateOptions() []string {
	return []string{"id", "ext", "title", "uploader", "channel", "upload_date", "duration_string"}
}

// mediaProfilesPresetOptions is preset_options/0.
func mediaProfilesPresetOptions() []CoreSelectOption {
	return []CoreSelectOption{
		{Label: "Default", Value: "default"},
		{Label: "Media Center (Plex, Jellyfin, Kodi, etc.)", Value: "media_center"},
		{Label: "Music", Value: "audio"},
		{Label: "Archiving", Value: "archiving"},
	}
}

// mediaProfilesDefaultOutputTemplate is default_output_template/0. Ecto sets
// schema field defaults on a bare struct literal, so this must go through
// the NewMediaProfile() constructor rather than a zero-value MediaProfile{}.
func mediaProfilesDefaultOutputTemplate() string {
	return store.NewMediaProfile().OutputPathTemplate
}

// mediaProfilesMediaCenterOutputTemplate is media_center_output_template/0.
func mediaProfilesMediaCenterOutputTemplate() string {
	return "/shows/{{ source_custom_name }}/{{ season_by_year__episode_by_date_and_index }} - {{ title }}.{{ ext }}"
}

// mediaProfilesAudioOutputTemplate is audio_output_template/0.
func mediaProfilesAudioOutputTemplate() string {
	return "/music/{{ artist_name }}/{{ title }}.{{ ext }}"
}

// double_brace/1 (Pinchflat.Utils.StringUtils), used throughout the form and
// output_template_help templates.
func mediaProfilesBrace(s string) string {
	return fsutil.DoubleBrace(s)
}
