package web

// Port of lib/pinchflat_web/components/layouts.ex. The renderable pieces
// (LayoutsNavLink, LayoutsFooterLink) are in layouts.templ; this file has
// active_path?/2 and the Settings/version lookups used by the layouts.

import (
	"context"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// layoutsApp is the serving App (nil outside a request, e.g. in component
// tests), carried on the Page.
func layoutsApp(ctx context.Context) *core.App { return PageOf(ctx).App }

// layoutsYtDlpVersion is Settings.get!(:yt_dlp_version), shown in the footer.
func layoutsYtDlpVersion(ctx context.Context) string {
	a := layoutsApp(ctx)
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
	a := layoutsApp(ctx)
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
func layoutsVersion(ctx context.Context) string { return PageOf(ctx).Version }

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
