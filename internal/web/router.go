package web

// Port of lib/pinchflat_web/router.ex. Every public URL is unchanged;
// /_live/* are new internal endpoints that replace LiveView events (htmx).
// Hand-written W4 infrastructure.

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Router builds the route table.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.NotFound(s.NotFound)
	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) { s.NotFound(w, req) })

	// scope "/" pipe_through [:maybe_basic_auth, :token_protected_route]
	r.Group(func(r chi.Router) {
		r.Use(s.maybeBasicAuth, s.tokenProtectedRoute)
		r.Get("/sources/opml", s.PodcastControllerOpmlFeed)
	})

	// scope "/" pipe_through :maybe_basic_auth
	r.Group(func(r chi.Router) {
		r.Use(s.maybeBasicAuth)
		r.Get("/sources/{uuid}/feed", s.PodcastControllerRssFeed)
		r.Get("/sources/{uuid}/feed_image", s.PodcastControllerFeedImage)
		r.Get("/media/{uuid}/episode_image", s.PodcastControllerEpisodeImage)
		r.Get("/media/{uuid}/stream", s.MediaItemControllerStream)
	})

	// scope "/" pipe_through :api
	r.Get("/healthcheck", s.HealthControllerCheck)

	// scope "/" pipe_through :browser
	r.Group(func(r chi.Router) {
		r.Use(s.basicAuth, secureBrowserHeaders, protectFromForgery)

		r.Get("/", s.PageControllerHome)

		// resources "/media_profiles"
		r.Get("/media_profiles", s.MediaProfileControllerIndex)
		r.Get("/media_profiles/new", s.MediaProfileControllerNew)
		r.Post("/media_profiles", s.MediaProfileControllerCreate)
		r.Get("/media_profiles/{id}", s.MediaProfileControllerShow)
		r.Get("/media_profiles/{id}/edit", s.MediaProfileControllerEdit)
		r.Patch("/media_profiles/{id}", s.MediaProfileControllerUpdate)
		r.Put("/media_profiles/{id}", s.MediaProfileControllerUpdate)
		r.Delete("/media_profiles/{id}", s.MediaProfileControllerDelete)

		r.Get("/search", s.SearchControllerShow)

		r.Get("/settings", s.SettingControllerShow)
		r.Patch("/settings", s.SettingControllerUpdate)
		r.Put("/settings", s.SettingControllerUpdate)
		r.Get("/app_info", s.SettingControllerAppInfo)
		r.Get("/download_logs", s.SettingControllerDownloadLogs)

		// resources "/sources" + nested actions
		r.Get("/sources", s.SourceControllerIndex)
		r.Get("/sources/new", s.SourceControllerNew)
		r.Post("/sources", s.SourceControllerCreate)
		r.Get("/sources/{id}", s.SourceControllerShow)
		r.Get("/sources/{id}/edit", s.SourceControllerEdit)
		r.Patch("/sources/{id}", s.SourceControllerUpdate)
		r.Put("/sources/{id}", s.SourceControllerUpdate)
		r.Delete("/sources/{id}", s.SourceControllerDelete)
		r.Post("/sources/{source_id}/force_download_pending", s.SourceControllerForceDownloadPending)
		r.Post("/sources/{source_id}/force_redownload", s.SourceControllerForceRedownload)
		r.Post("/sources/{source_id}/force_index", s.SourceControllerForceIndex)
		r.Post("/sources/{source_id}/force_metadata_refresh", s.SourceControllerForceMetadataRefresh)
		r.Post("/sources/{source_id}/sync_files_on_disk", s.SourceControllerSyncFilesOnDisk)

		// resources "/media", only: [:show, :edit, :update, :delete], nested under sources
		r.Get("/sources/{source_id}/media/{id}", s.MediaItemControllerShow)
		r.Get("/sources/{source_id}/media/{id}/edit", s.MediaItemControllerEdit)
		r.Patch("/sources/{source_id}/media/{id}", s.MediaItemControllerUpdate)
		r.Put("/sources/{source_id}/media/{id}", s.MediaItemControllerUpdate)
		r.Delete("/sources/{source_id}/media/{id}", s.MediaItemControllerDelete)
		r.Post("/sources/{source_id}/media/{media_item_id}/force_download", s.MediaItemControllerForceDownload)

		// LiveView replacements (htmx fragments and actions)
		r.Get("/_live/jobs", s.JobTableLiveRender)
		r.Get("/_live/history", s.HistoryTableLiveRender)
		r.Get("/_live/sources", s.SourceLiveIndexTableLiveRender)
		r.Post("/_live/sources/{id}/enabled", s.SourceEnableToggleUpdate)
		r.Get("/_live/sources/{id}/media", s.MediaItemTableLiveRender)
		r.Post("/_live/upgrade", s.UpgradeButtonLiveCheckMatchingText)
	})

	return r
}

// URLParam returns a route parameter ({id}, {uuid}, ...).
func URLParam(r *http.Request, name string) string { return chi.URLParam(r, name) }
