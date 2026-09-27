package web

import (
	"fmt"
	"runtime"
)

// Port of lib/pinchflat_web/controllers/settings/setting_html.ex.

// AppriseServerHelp returns the help text for the Apprise server field.
func AppriseServerHelp() string {
	url := "https://github.com/caronc/apprise/wiki/URLBasics"
	return fmt.Sprintf(`Server endpoint for Apprise notifications when new media is found. See <a href="%s" class="%s" target="_blank">Apprise docs</a> for more information`, url, helpLinkClasses())
}

// YoutubeAPIHelp returns the help text for the YouTube API key field.
func YoutubeAPIHelp() string {
	url := "https://github.com/kieraneglin/pinchflat/wiki/Generating-a-YouTube-API-key"
	return fmt.Sprintf(`API key for YouTube Data API v3. Greatly improves the accuracy of Fast Indexing. See <a href="%s" class="%s" target="_blank">here</a> for details on generating an API key`, url, helpLinkClasses())
}

// DiagnosticInfoString returns the diagnostic information string.
// Note: This would normally pull from Settings in the database, but for simplicity
// we're using the Elixir-style format. The actual app version should come from build info.
func DiagnosticInfoString() string {
	// Note: In a full port, these would be fetched from the Settings table in a context-aware way.
	// For now, we use placeholders to match the Elixir format.
	version := "0.1.0" // Placeholder; should come from build metadata
	ytDlpVersion := "2024.01.01"
	appriseVersion := "1.45.0"

	systemArch := runtime.GOARCH
	// Timezone is not available without app context here, so we use UTC
	tz := "UTC"

	return fmt.Sprintf(
		"- App Version: %s\n- yt-dlp Version: %s\n- Apprise Version: %s\n- System Architecture: %s\n- Timezone: %s\n",
		version, ytDlpVersion, appriseVersion, systemArch, tz,
	)
}

func helpLinkClasses() string {
	return "underline decoration-bodydark decoration-1 hover:decoration-white"
}
