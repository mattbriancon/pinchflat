package store

import (
	"net/url"
)

type Setting struct {
	ID                            int64   `db:"id"`
	Onboarding                    bool    `db:"onboarding"`
	YtDlpVersion                  *string `db:"yt_dlp_version"`
	VideoCodecPreference          string  `db:"video_codec_preference"`
	AudioCodecPreference          string  `db:"audio_codec_preference"`
	YoutubeAPIKey                 *string `db:"youtube_api_key"`
	RouteToken                    string  `db:"route_token"`
	ExtractorSleepIntervalSeconds int     `db:"extractor_sleep_interval_seconds"`
	DownloadThroughputLimit       *string `db:"download_throughput_limit"`
	RestrictFilenames             *bool   `db:"restrict_filenames"`
}

func (Setting) TableName() string { return "settings" }

// SettingParams are the typed parameters for updating a Setting.
// Nil pointer fields mean the value was not submitted.
type SettingParams struct {
	Onboarding                    *bool
	YtDlpVersion                  *string
	VideoCodecPreference          *string
	AudioCodecPreference          *string
	YoutubeAPIKey                 *string
	ExtractorSleepIntervalSeconds *int
	DownloadThroughputLimit       *string
	RestrictFilenames             *bool
}

// ParseSettingParams reads the "setting[...]" fields of the settings form.
func ParseSettingParams(values url.Values) (SettingParams, map[string][]string) {
	f := newFormReader(values, "setting")
	p := SettingParams{}
	f.boolOrFalse("onboarding", &p.Onboarding)
	f.str("yt_dlp_version", &p.YtDlpVersion)
	f.str("video_codec_preference", &p.VideoCodecPreference)
	f.str("audio_codec_preference", &p.AudioCodecPreference)
	f.str("youtube_api_key", &p.YoutubeAPIKey)
	f.str("download_throughput_limit", &p.DownloadThroughputLimit)
	f.boolOrFalse("restrict_filenames", &p.RestrictFilenames)
	switch n, state := f.integer("extractor_sleep_interval_seconds"); state {
	case fieldSet:
		p.ExtractorSleepIntervalSeconds = &n
	case fieldBlank:
		addErr(f.errs, "extractor_sleep_interval_seconds", "can't be blank")
	}
	return p, f.errs
}

// Validate is SettingChangeset's validate_required + validate_number: the
// required columns must be non-blank after p is applied, and a submitted
// sleep interval must not be negative.
func (p SettingParams) Validate(existing *Setting) map[string][]string {
	errs := map[string][]string{}
	next, _ := p.apply(existing)
	if isBlank(next.VideoCodecPreference) {
		addErr(errs, "video_codec_preference", "can't be blank")
	}
	if isBlank(next.AudioCodecPreference) {
		addErr(errs, "audio_codec_preference", "can't be blank")
	}
	if p.ExtractorSleepIntervalSeconds != nil && *p.ExtractorSleepIntervalSeconds < 0 {
		addErr(errs, "extractor_sleep_interval_seconds", "must be greater than or equal to 0")
	}
	return errs
}

// apply returns a copy of existing with p applied and the columns that
// changed.
func (p SettingParams) apply(existing *Setting) (*Setting, changes) {
	next := *existing
	c := changes{}
	setValue(c, "onboarding", &next.Onboarding, p.Onboarding)
	setString(c, "video_codec_preference", &next.VideoCodecPreference, p.VideoCodecPreference)
	setString(c, "audio_codec_preference", &next.AudioCodecPreference, p.AudioCodecPreference)
	setValue(c, "extractor_sleep_interval_seconds", &next.ExtractorSleepIntervalSeconds, p.ExtractorSleepIntervalSeconds)
	setNullableString(c, "yt_dlp_version", &next.YtDlpVersion, p.YtDlpVersion, false)
	setNullableString(c, "youtube_api_key", &next.YoutubeAPIKey, p.YoutubeAPIKey, false)
	setNullableString(c, "download_throughput_limit", &next.DownloadThroughputLimit, p.DownloadThroughputLimit, false)
	setNullable(c, "restrict_filenames", &next.RestrictFilenames, p.RestrictFilenames, false)
	return &next, c
}
