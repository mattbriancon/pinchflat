package web

import (
	"context"
	"net/http"
	"strconv"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// PageControllerHome: home action
func (s *Server) PageControllerHome(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	onboardingParam := r.URL.Query().Get("onboarding")
	doneOnboarding := onboardingParam == "0"
	forceOnboarding := onboardingParam == "1"

	if doneOnboarding {
		if _, err := s.App.SetSetting(ctx, store.KW{store.Opt("onboarding", false)}); err != nil {
			s.Fail(w, r, err)
			return
		}
	}

	onboarding, _ := s.App.GetSettingBang(ctx, "onboarding")
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
	mediaProfileCount, err := store.Scalar[int](ctx, s.App.Q(ctx),
		store.SQ.Select("COUNT(*)").From("media_profiles"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Get source count
	sourceCount, err := store.Scalar[int](ctx, s.App.Q(ctx),
		store.SQ.Select("COUNT(*)").From("sources"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Get downloaded media items stats
	downloadedMediaItems := store.MediaQueryNew().Where(store.MediaQueryDownloaded())
	mediaItemSize, err := store.Scalar[int](ctx, s.App.Q(ctx),
		store.SQ.Select("COALESCE(SUM(mi.media_size_bytes), 0)").FromSelect(downloadedMediaItems.B, "mi"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	mediaItemCount, err := store.Scalar[int](ctx, s.App.Q(ctx),
		store.SQ.Select("COUNT(*)").FromSelect(downloadedMediaItems.B, "mi"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	downloadedRecords, downloadedPage, downloadedTotalPages, downloadedTotalCount, err := historyTableFetch(
		ctx, s.App, "downloaded", queryPage(r, "downloaded_page"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	pendingRecords, pendingPage, pendingTotalPages, pendingTotalCount, err := historyTableFetch(
		ctx, s.App, "pending", queryPage(r, "pending_page"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	tasks, err := getJobTableTasks(ctx, s.App)
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
		downloadedRecords, downloadedPage, downloadedTotalPages, downloadedTotalCount,
		pendingRecords, pendingPage, pendingTotalPages, pendingTotalCount,
		tasks,
	))
}

// queryPage reads a page-number query param, defaulting to 1.
func queryPage(r *http.Request, name string) int {
	page, _ := strconv.Atoi(r.URL.Query().Get(name))
	if page < 1 {
		return 1
	}
	return page
}

// renderOnboardingPage renders the onboarding checklist page.
func renderOnboardingPage(s *Server, ctx context.Context, w http.ResponseWriter, r *http.Request) {
	if _, err := s.App.SetSetting(ctx, store.KW{store.Opt("onboarding", true)}); err != nil {
		s.Fail(w, r, err)
		return
	}

	// Check if media profiles exist
	mediaProfileCount, err := store.Scalar[int](ctx, s.App.Q(ctx),
		store.SQ.Select("COUNT(*)").From("media_profiles"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	mediaProfilesExist := mediaProfileCount > 0

	// Check if sources exist
	sourceCount, err := store.Scalar[int](ctx, s.App.Q(ctx),
		store.SQ.Select("COUNT(*)").From("sources"))
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
