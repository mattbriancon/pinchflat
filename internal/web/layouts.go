package web

// Port of lib/pinchflat_web/components/layouts.ex. The renderable pieces
// (LayoutsNavLink, LayoutsFooterLink) are in layouts.templ; this file has
// active_path?/2 and the process-wide App/Options accessors described in
// server.go's New.

import (
	"context"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
)

var (
	globalApp  *core.App
	globalOpts Options
)

// currentApp is the App set by the last web.New call. Safe for the current
// test suite (one *Server built at a time, per t.Run), but a real second
// caller would race; see the report.
func currentApp() *core.App { return globalApp }

// currentOpts is the Options set by the last web.New call (currentOpts().Version).
func currentOpts() Options { return globalOpts }

// layoutsYtDlpVersion is Settings.get!(:yt_dlp_version), shown in the footer.
func layoutsYtDlpVersion(ctx context.Context) string {
	a := currentApp()
	if a == nil {
		return ""
	}
	v, err := a.SettingsGetBang(ctx, "yt_dlp_version")
	if err != nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// layoutsProEnabled is Settings.get!(:pro_enabled), used by the root layout's
// x-data (whether the upgrade modal can show).
func layoutsProEnabled(ctx context.Context) bool {
	a := currentApp()
	if a == nil {
		return false
	}
	v, err := a.SettingsGet(ctx, "pro_enabled")
	if err != nil {
		return false
	}
	b, _ := v.(bool)
	return b
}

// layoutsVersion is Application.spec(:pinchflat)[:vsn].
func layoutsVersion() string { return currentOpts().Version }

// layoutsActivePath is active_path?/2.
func layoutsActivePath(href, currentPath string) bool {
	if currentPath == "" {
		return false
	}
	if href == currentPath {
		return true
	}
	if href == "/" {
		return false
	}
	return strings.HasPrefix(currentPath, href+"/")
}
