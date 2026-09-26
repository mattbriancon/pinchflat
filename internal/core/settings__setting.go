package core

type Setting struct {
	ID                            int64   `db:"id"`
	Onboarding                    bool    `db:"onboarding"`
	ProEnabled                    bool    `db:"pro_enabled"`
	YtDlpVersion                  *string `db:"yt_dlp_version"`
	AppriseServer                 *string `db:"apprise_server"`
	AppriseVersion                *string `db:"apprise_version"`
	VideoCodecPreference          string  `db:"video_codec_preference"`
	AudioCodecPreference          string  `db:"audio_codec_preference"`
	YoutubeAPIKey                 *string `db:"youtube_api_key"`
	RouteToken                    string  `db:"route_token"`
	ExtractorSleepIntervalSeconds int     `db:"extractor_sleep_interval_seconds"`
	DownloadThroughputLimit       *string `db:"download_throughput_limit"`
	RestrictFilenames             *bool   `db:"restrict_filenames"`
}

func (Setting) TableName() string { return "settings" }

func NewSetting() *Setting {
	return &Setting{
		Onboarding:                    true,
		ProEnabled:                    false,
		VideoCodecPreference:          "avc",
		AudioCodecPreference:          "m4a",
		RouteToken:                    "tmp-token",
		ExtractorSleepIntervalSeconds: 0,
		RestrictFilenames:             Ptr(false),
	}
}

var settingAllowedFields = []string{
	"onboarding",
	"pro_enabled",
	"yt_dlp_version",
	"apprise_version",
	"apprise_server",
	"video_codec_preference",
	"audio_codec_preference",
	"youtube_api_key",
	"extractor_sleep_interval_seconds",
	"download_throughput_limit",
	"restrict_filenames",
}

var settingRequiredFields = []string{
	"onboarding",
	"pro_enabled",
	"video_codec_preference",
	"audio_codec_preference",
	"extractor_sleep_interval_seconds",
}

// SettingChangeset/3
func SettingChangeset(setting *Setting, attrs Attrs) *Changeset {
	panic("unported: Pinchflat.Settings.Setting.changeset/2")
}
