package web

// Every public URL is unchanged. There are no htmx fragment routes: every
// page renders everything on first load, and the two mutations that used
// to be LiveView events are plain form POSTs that redirect back.

import "net/http"

// Router builds the route table.
func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	handle := func(pattern string, h http.HandlerFunc, mw ...func(http.Handler) http.Handler) {
		var handler http.Handler = h
		for i := len(mw) - 1; i >= 0; i-- {
			handler = mw[i](handler)
		}
		mux.Handle(pattern, handler)
	}

	// Anything unmatched, including a known path with the wrong method, is a 404.
	handle("/", s.NotFound)

	// scope "/" pipe_through [:maybe_basic_auth, :token_protected_route]
	handle("GET /sources/opml", s.PodcastControllerOpmlFeed, s.maybeBasicAuth, s.tokenProtectedRoute)

	// scope "/" pipe_through :maybe_basic_auth
	for pattern, h := range map[string]http.HandlerFunc{
		"GET /sources/{uuid}/feed":        s.PodcastControllerRssFeed,
		"GET /sources/{uuid}/feed_image":  s.PodcastControllerFeedImage,
		"GET /media/{uuid}/episode_image": s.PodcastControllerEpisodeImage,
		"GET /media/{uuid}/stream":        s.MediaItemControllerStream,
	} {
		handle(pattern, h, s.maybeBasicAuth)
	}

	// scope "/" pipe_through :api
	handle("GET /healthcheck", s.HealthControllerCheck)

	// scope "/" pipe_through :browser
	for pattern, h := range map[string]http.HandlerFunc{
		"GET /{$}": s.PageControllerHome,

		// resources "/media_profiles"
		"GET /media_profiles":           s.MediaProfileControllerIndex,
		"GET /media_profiles/new":       s.MediaProfileControllerNew,
		"POST /media_profiles":          s.MediaProfileControllerCreate,
		"GET /media_profiles/{id}":      s.MediaProfileControllerShow,
		"GET /media_profiles/{id}/edit": s.MediaProfileControllerEdit,
		"PATCH /media_profiles/{id}":    s.MediaProfileControllerUpdate,
		"PUT /media_profiles/{id}":      s.MediaProfileControllerUpdate,
		"DELETE /media_profiles/{id}":   s.MediaProfileControllerDelete,

		"GET /search": s.SearchControllerShow,

		"GET /settings":      s.SettingControllerShow,
		"PATCH /settings":    s.SettingControllerUpdate,
		"PUT /settings":      s.SettingControllerUpdate,
		"GET /app_info":      s.SettingControllerAppInfo,
		"GET /download_logs": s.SettingControllerDownloadLogs,

		// resources "/sources" + nested actions
		"GET /sources":           s.SourceControllerIndex,
		"GET /sources/new":       s.SourceControllerNew,
		"POST /sources":          s.SourceControllerCreate,
		"GET /sources/{id}":      s.SourceControllerShow,
		"GET /sources/{id}/edit": s.SourceControllerEdit,
		"PATCH /sources/{id}":    s.SourceControllerUpdate,
		"PUT /sources/{id}":      s.SourceControllerUpdate,
		"DELETE /sources/{id}":   s.SourceControllerDelete,
		"POST /sources/{source_id}/force_download_pending": s.SourceControllerForceDownloadPending,
		"POST /sources/{source_id}/force_redownload":       s.SourceControllerForceRedownload,
		"POST /sources/{source_id}/force_index":            s.SourceControllerForceIndex,
		"POST /sources/{source_id}/force_metadata_refresh": s.SourceControllerForceMetadataRefresh,
		"POST /sources/{source_id}/sync_files_on_disk":     s.SourceControllerSyncFilesOnDisk,
		"POST /sources/{id}/enabled":                       s.SourceEnableToggleUpdate,

		// resources "/media", only: [:show, :edit, :update, :delete], nested under sources
		"GET /sources/{source_id}/media/{id}":                            s.MediaItemControllerShow,
		"GET /sources/{source_id}/media/{id}/edit":                       s.MediaItemControllerEdit,
		"PATCH /sources/{source_id}/media/{id}":                          s.MediaItemControllerUpdate,
		"PUT /sources/{source_id}/media/{id}":                            s.MediaItemControllerUpdate,
		"DELETE /sources/{source_id}/media/{id}":                         s.MediaItemControllerDelete,
		"POST /sources/{source_id}/media/{media_item_id}/force_download": s.MediaItemControllerForceDownload,

		"POST /tasks/{id}/retry": s.TaskControllerRetry,
	} {
		handle(pattern, h, s.basicAuth, secureBrowserHeaders, protectFromForgery)
	}

	return mux
}
