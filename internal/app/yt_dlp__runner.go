package app

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/ytdlp"
)

// NewYtDlpRunner builds the real yt-dlp runner from a's config, reading
// per-call settings (download_throughput_limit, extractor_sleep_interval_seconds,
// restrict_filenames) through a.GetSettingBang.
func NewYtDlpRunner(a *App) *ytdlp.Runner {
	return &ytdlp.Runner{
		Executable:   a.Config.YtDlpExecutable,
		TmpDir:       a.Config.TmpfileDirectory,
		CookieDir:    a.Config.ExtrasDirectory,
		SettingsFunc: func(ctx context.Context) ytdlp.Settings { return ytDlpSettingsFor(ctx, a) },
	}
}

func ytDlpSettingsFor(ctx context.Context, a *App) ytdlp.Settings {
	var settings ytdlp.Settings

	if v, err := a.GetSettingBang(ctx, "download_throughput_limit"); err == nil {
		if s, ok := v.(string); ok {
			settings.ThroughputLimit = s
		}
	}
	if v, err := a.GetSettingBang(ctx, "extractor_sleep_interval_seconds"); err == nil {
		if n, ok := v.(int); ok {
			settings.SleepIntervalSeconds = n
		}
	}
	if v, err := a.GetSettingBang(ctx, "restrict_filenames"); err == nil {
		if b, ok := v.(bool); ok {
			settings.RestrictFilenames = b
		}
	}

	return settings
}
