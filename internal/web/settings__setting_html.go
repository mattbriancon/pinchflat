package web

import (
	"context"
	"fmt"
	"runtime"
)

// Port of lib/pinchflat_web/controllers/settings/setting_html.ex.

// YoutubeAPIHelp returns the help text for the YouTube API key field.
func YoutubeAPIHelp() string {
	url := "https://github.com/kieraneglin/pinchflat/wiki/Generating-a-YouTube-API-key"
	return fmt.Sprintf(`API key for YouTube Data API v3. Greatly improves the accuracy of Fast Indexing. See <a href="%s" class="%s" target="_blank">here</a> for details on generating an API key`, url, helpLinkClasses())
}

// DiagnosticInfoString returns the diagnostic information string
// (diagnostic_info_string/0). appVersion is Application.spec(:pinchflat)[:vsn]
// (the running server's Options.Version).
func DiagnosticInfoString(ctx context.Context, appVersion string) string {
	page := PageOf(ctx)

	ytDlpVersion, _ := page.App.SettingsGetBang(ctx, "yt_dlp_version")
	systemArch := runtime.GOOS + "-" + runtime.GOARCH

	return fmt.Sprintf(
		"- App Version: %s\n- yt-dlp Version: %v\n- System Architecture: %s\n- Timezone: %s\n",
		appVersion, ytDlpVersion, systemArch, page.App.Config.Timezone,
	)
}

func helpLinkClasses() string {
	return "underline decoration-bodydark decoration-1 hover:decoration-white"
}
