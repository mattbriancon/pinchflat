package web

// The renderable pieces (LayoutsNavLink, LayoutsFooterLink) are in
// layouts.templ; this file has the active-path check and the
// Settings/version lookups used by the layouts.

import (
	"context"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/app"
)

// layoutsApp is the serving App (nil outside a request, e.g. in component
// tests), carried on the Page.
func layoutsApp(ctx context.Context) *app.App { return PageOf(ctx).App }

// layoutsYtDlpVersion is Settings.get!(:yt_dlp_version), shown in the footer.
func layoutsYtDlpVersion(ctx context.Context) string {
	a := layoutsApp(ctx)
	if a == nil {
		return ""
	}
	v, err := a.GetSettingBang(ctx, "yt_dlp_version")
	if err != nil {
		return ""
	}
	s, _ := v.(string)
	return s
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
