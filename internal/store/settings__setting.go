package store

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

var settingAllowedFields = []string{
	"onboarding",
	"yt_dlp_version",
	"video_codec_preference",
	"audio_codec_preference",
	"youtube_api_key",
	"extractor_sleep_interval_seconds",
	"download_throughput_limit",
	"restrict_filenames",
}

var settingRequiredFields = []string{
	"onboarding",
	"video_codec_preference",
	"audio_codec_preference",
	"extractor_sleep_interval_seconds",
}

// SettingChangeset/2
func SettingChangeset(setting *Setting, attrs Attrs) *Changeset {
	return Cast(setting, attrs, settingAllowedFields).
		ValidateRequired(settingRequiredFields...).
		ValidateNumber("extractor_sleep_interval_seconds", NumberOpts{
			GreaterThanOrEqualTo: Num(0),
		})
}
