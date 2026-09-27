package web

import (
	"context"
	"net/http"

	"github.com/mattbriancon/pinchflat/internal/core"
)

// Port of lib/pinchflat_web/controllers/pages/page_controller.ex.

// PageControllerHome: home action
func (s *Server) PageControllerHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	onboardingParam := r.URL.Query().Get("onboarding")
	doneOnboarding := onboardingParam == "0"
	forceOnboarding := onboardingParam == "1"

	if doneOnboarding {
		if _, err := s.App.SettingsSet(ctx, core.KW{core.Opt("onboarding", false)}); err != nil {
			s.Fail(w, r, err)
			return
		}
	}

	onboarding, _ := s.App.SettingsGetBang(ctx, "onboarding")
	forceOnboarding = forceOnboarding || onboarding.(bool)

	if forceOnboarding {
		renderOnboardingPage(s, ctx, w, r)
	} else {
		renderHomePage(s, ctx, w, r)
	}
}

// renderHomePage renders the home page with stats.
func renderHomePage(s *Server, ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Get media profile count
	mediaProfileCount, err := core.Scalar[int](ctx, s.App.Q(ctx),
		core.SQ.Select("COUNT(*)").From("media_profiles"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Get source count
	sourceCount, err := core.Scalar[int](ctx, s.App.Q(ctx),
		core.SQ.Select("COUNT(*)").From("sources"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Get downloaded media items stats
	downloadedMediaItems := core.MediaQueryNew().Where(core.MediaQueryDownloaded())
	mediaItemSize, err := core.Scalar[int](ctx, s.App.Q(ctx),
		core.SQ.Select("COALESCE(SUM(mi.media_size_bytes), 0)").FromSelect(downloadedMediaItems.B, "mi"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	mediaItemCount, err := core.Scalar[int](ctx, s.App.Q(ctx),
		core.SQ.Select("COUNT(*)").FromSelect(downloadedMediaItems.B, "mi"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	s.Render(w, r, http.StatusOK, LayoutApp, PagesPageHTMLHome(
		ctx,
		int64(mediaProfileCount),
		int64(sourceCount),
		int64(mediaItemCount),
		int64(mediaItemSize),
	))
}

// renderOnboardingPage renders the onboarding checklist page.
func renderOnboardingPage(s *Server, ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if _, err := s.App.SettingsSet(ctx, core.KW{core.Opt("onboarding", true)}); err != nil {
		s.Fail(w, r, err)
		return
	}

	// Check if media profiles exist
	mediaProfileCount, err := core.Scalar[int](ctx, s.App.Q(ctx),
		core.SQ.Select("COUNT(*)").From("media_profiles"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	mediaProfilesExist := mediaProfileCount > 0

	// Check if sources exist
	sourceCount, err := core.Scalar[int](ctx, s.App.Q(ctx),
		core.SQ.Select("COUNT(*)").From("sources"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	sourcesExist := sourceCount > 0

	s.Render(w, r, http.StatusOK, LayoutOnboarding, PagesPageHTMLOnboardingChecklist(
		ctx,
		mediaProfilesExist,
		sourcesExist,
	))
}
