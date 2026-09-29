package store

import (
	"net/url"
	"strconv"
	"strings"
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

// ParseSettingParams parses form values into SettingParams, returning validation errors.
func ParseSettingParams(values url.Values) (SettingParams, map[string][]string) {
	errs := make(map[string][]string)
	p := SettingParams{}

	if v := values.Get("onboarding"); v != "" {
		b := v == "true" || v == "on" || v == "1"
		p.Onboarding = &b
	}

	if v := values.Get("yt_dlp_version"); v != "" {
		v = strings.TrimSpace(v)
		if v == "" {
			v = ""
		}
		p.YtDlpVersion = &v
	}

	if v := values.Get("video_codec_preference"); v != "" {
		v = strings.TrimSpace(v)
		p.VideoCodecPreference = &v
	}

	if v := values.Get("audio_codec_preference"); v != "" {
		v = strings.TrimSpace(v)
		p.AudioCodecPreference = &v
	}

	if v := values.Get("youtube_api_key"); v != "" {
		v = strings.TrimSpace(v)
		if v == "" {
			v = ""
		}
		p.YoutubeAPIKey = &v
	}

	if v := values.Get("extractor_sleep_interval_seconds"); v != "" {
		i, err := strconv.Atoi(v)
		if err != nil {
			errs["extractor_sleep_interval_seconds"] = append(errs["extractor_sleep_interval_seconds"], "is not a valid integer")
		} else {
			p.ExtractorSleepIntervalSeconds = &i
		}
	}

	if v := values.Get("download_throughput_limit"); v != "" {
		v = strings.TrimSpace(v)
		if v == "" {
			v = ""
		}
		p.DownloadThroughputLimit = &v
	}

	if v := values.Get("restrict_filenames"); v != "" {
		b := v == "true" || v == "on" || v == "1"
		p.RestrictFilenames = &b
	}

	return p, errs
}

// Validate validates the SettingParams against an existing Setting and returns a map of field errors.
// The existing Setting is used to check for required fields when they're provided.
func (p SettingParams) Validate(existing *Setting) map[string][]string {
	errs := make(map[string][]string)

	// Check required fields: onboarding, video_codec_preference, audio_codec_preference, extractor_sleep_interval_seconds
	if p.Onboarding == nil && !existing.Onboarding {
		errs["onboarding"] = append(errs["onboarding"], "can't be blank")
	}

	if p.VideoCodecPreference == nil && existing.VideoCodecPreference == "" {
		errs["video_codec_preference"] = append(errs["video_codec_preference"], "can't be blank")
	} else if p.VideoCodecPreference != nil && *p.VideoCodecPreference == "" {
		errs["video_codec_preference"] = append(errs["video_codec_preference"], "can't be blank")
	}

	if p.AudioCodecPreference == nil && existing.AudioCodecPreference == "" {
		errs["audio_codec_preference"] = append(errs["audio_codec_preference"], "can't be blank")
	} else if p.AudioCodecPreference != nil && *p.AudioCodecPreference == "" {
		errs["audio_codec_preference"] = append(errs["audio_codec_preference"], "can't be blank")
	}

	if p.ExtractorSleepIntervalSeconds == nil && existing.ExtractorSleepIntervalSeconds < 0 {
		errs["extractor_sleep_interval_seconds"] = append(errs["extractor_sleep_interval_seconds"], "can't be blank")
	} else if p.ExtractorSleepIntervalSeconds != nil && *p.ExtractorSleepIntervalSeconds < 0 {
		errs["extractor_sleep_interval_seconds"] = append(errs["extractor_sleep_interval_seconds"], "must be greater than or equal to 0")
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
