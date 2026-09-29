package web

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// sourceForm builds the source form: source's values, overlaid with the raw
// submitted form when redisplaying after a failed submit.
func sourceForm(source *store.Source, submitted url.Values, errs map[string][]string) *Form {
	values := map[string]any{
		"enabled":                       source.Enabled,
		"custom_name":                   source.CustomName,
		"original_url":                  source.OriginalURL,
		"media_profile_id":              source.MediaProfileID,
		"index_frequency_minutes":       source.IndexFrequencyMinutes,
		"fast_index":                    source.FastIndex,
		"download_media":                source.DownloadMedia,
		"cookie_behaviour":              source.CookieBehaviour,
		"download_cutoff_date":          source.DownloadCutoffDate,
		"retention_period_days":         source.RetentionPeriodDays,
		"min_duration_seconds":          source.MinDurationSeconds,
		"max_duration_seconds":          source.MaxDurationSeconds,
		"title_filter_regex":            source.TitleFilterRegex,
		"output_path_template_override": source.OutputPathTemplateOverride,
	}
	for field := range values {
		if vs := submitted["source["+field+"]"]; len(vs) > 0 {
			values[field] = vs[len(vs)-1]
		}
	}
	return NewForm("source", values, errs)
}

// SourceControllerIndex renders the sources index page.
func (s *Server) SourceControllerIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	state, err := s.sourceIndexTableFetch(ctx, r)
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	s.Render(w, r, http.StatusOK, LayoutApp, SourceHTMLIndex(state))
}

// SourceControllerNew renders the new source form.
func (s *Server) SourceControllerNew(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	templateIDStr := r.URL.Query().Get("template_id")

	// Get the template source if provided
	var cs *store.Source
	if templateID, err := strconv.ParseInt(templateIDStr, 10, 64); err == nil && templateID > 0 {
		if source, err := s.App.GetSource(ctx, templateID); err == nil {
			cs = source
		}
	}
	if cs == nil {
		cs = &store.Source{}
	}

	// Create a blank source with nullified fields from the template
	source := &store.Source{
		IndexFrequencyMinutes:      cs.IndexFrequencyMinutes,
		DownloadCutoffDate:         cs.DownloadCutoffDate,
		TitleFilterRegex:           cs.TitleFilterRegex,
		OutputPathTemplateOverride: cs.OutputPathTemplateOverride,
		CookieBehaviour:            cs.CookieBehaviour,
		MediaProfileID:             cs.MediaProfileID,
	}

	mediaProfiles, _ := s.App.ListMediaProfiles(ctx)

	layout := OnboardingLayout(ctx)
	s.Render(w, r, http.StatusOK, layout, SourceHTMLNew(sourceForm(source, nil, nil), mediaProfiles))
}

// SourceControllerCreate creates a new source.
func (s *Server) SourceControllerCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	_ = r.ParseForm()

	source, err := s.App.SourcesCreateSource(ctx, store.ParseSourceParams(r.PostForm), true)
	if err != nil {
		if errs, ok := store.AsValidationErrors(err); ok {
			mediaProfiles, _ := s.App.ListMediaProfiles(ctx)
			layout := OnboardingLayout(ctx)
			s.Render(w, r, http.StatusOK, layout, SourceHTMLNew(sourceForm(store.NewSource(), r.PostForm, errs), mediaProfiles))
			return
		}
		s.Fail(w, r, err)
		return
	}

	// Determine redirect location
	onboarding, _ := s.App.GetSetting(ctx, "onboarding")
	redirectPath := P(ctx, "/sources/%v", source.ID)
	if onboarding == true {
		redirectPath = P(ctx, "/?onboarding=1")
	}

	s.PutFlash(w, r, "info", "Source created successfully.")
	s.Redirect(w, r, redirectPath)
}

// SourceControllerShow displays a source.
func (s *Server) SourceControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	source, ok := loadOrFail(s, w, r, "id", s.App.GetSource)
	if !ok {
		return
	}

	source, _ = s.App.PreloadSourceMediaProfile(ctx, source)

	pendingTasks, _ := s.App.ListTasksFor(ctx, source, nil, []string{"executing", "available", "scheduled", "retryable"})
	for i, task := range pendingTasks {
		pendingTasks[i], _ = s.App.PreloadTaskJob(ctx, task)
	}

	pending, err := s.mediaItemTableFetch(ctx, r, source, "pending")
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	downloaded, err := s.mediaItemTableFetch(ctx, r, source, "downloaded")
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	other, err := s.mediaItemTableFetch(ctx, r, source, "other")
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	s.Render(w, r, http.StatusOK, LayoutApp, SourceHTMLShow(source, pendingTasks, pending, downloaded, other))
}

// SourceControllerEdit renders the source edit form.
func (s *Server) SourceControllerEdit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	source, ok := loadOrFail(s, w, r, "id", s.App.GetSource)
	if !ok {
		return
	}

	mediaProfiles, _ := s.App.ListMediaProfiles(ctx)

	s.Render(w, r, http.StatusOK, LayoutApp, SourceHTMLEdit(source, sourceForm(source, nil, nil), mediaProfiles))
}

// SourceControllerUpdate updates a source.
func (s *Server) SourceControllerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	source, ok := loadOrFail(s, w, r, "id", s.App.GetSource)
	if !ok {
		return
	}
	_ = r.ParseForm()

	updated, err := s.App.SourcesUpdateSource(ctx, source, store.ParseSourceParams(r.PostForm), true)
	if err != nil {
		if errs, ok := store.AsValidationErrors(err); ok {
			// Re-render form with errors, passing the original loaded source
			mediaProfiles, _ := s.App.ListMediaProfiles(ctx)
			s.Render(w, r, http.StatusOK, LayoutApp, SourceHTMLEdit(source, sourceForm(source, r.PostForm, errs), mediaProfiles))
			return
		}
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "Source updated successfully.")
	s.Redirect(w, r, P(ctx, "/sources/%v", updated.ID))
}

// SourceControllerDelete marks a source for deletion.
func (s *Server) SourceControllerDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	source, ok := loadOrFail(s, w, r, "id", s.App.GetSource)
	if !ok {
		return
	}
	deleteFiles := r.URL.Query().Get("delete_files") == "true"

	// Mark for deletion
	_, err := s.App.SourcesUpdateSource(ctx, source, store.SourceParams{
		MarkedForDeletionAt: store.Ptr(time.Now().UTC()),
	}, true)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Kickoff deletion worker
	_, _ = s.App.SourceDeletionWorkerKickoff(ctx, source, map[string]any{
		"delete_files": deleteFiles,
	})

	s.PutFlash(w, r, "info", "Source deletion started. This may take a while to complete.")
	s.Redirect(w, r, P(ctx, "/sources"))
}

// sourceForceAction loads the source named by the route's {source_id}, runs
// do against it, then flashes msg and redirects to the source's page. It
// factors out the shared shape of the "force X" actions below.
func (s *Server) sourceForceAction(w http.ResponseWriter, r *http.Request, msg string, do func(ctx context.Context, source *store.Source) error) {
	source, ok := loadOrFail(s, w, r, "source_id", s.App.GetSource)
	if !ok {
		return
	}
	if err := do(r.Context(), source); err != nil {
		s.Fail(w, r, err)
		return
	}
	s.PutFlash(w, r, "info", msg)
	s.Redirect(w, r, P(r.Context(), "/sources/%v", source.ID))
}

// SourceControllerForceDownloadPending forces pending media downloads.
func (s *Server) SourceControllerForceDownloadPending(w http.ResponseWriter, r *http.Request) {
	s.sourceForceAction(w, r, "Forcing download of pending media items.", func(ctx context.Context, source *store.Source) error {
		return s.App.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, nil)
	})
}

// SourceControllerForceRedownload forces redownload of existing media.
func (s *Server) SourceControllerForceRedownload(w http.ResponseWriter, r *http.Request) {
	s.sourceForceAction(w, r, "Forcing re-download of downloaded media items.", func(ctx context.Context, source *store.Source) error {
		_, err := s.App.DownloadingHelpersKickoffRedownloadForExistingMedia(ctx, source)
		return err
	})
}

// SourceControllerForceIndex forces an indexing task.
func (s *Server) SourceControllerForceIndex(w http.ResponseWriter, r *http.Request) {
	s.sourceForceAction(w, r, "Index enqueued.", func(ctx context.Context, source *store.Source) error {
		_, err := s.App.SlowIndexingHelpersKickoffIndexingTask(ctx, source, map[string]any{"force": true})
		return err
	})
}

// SourceControllerForceMetadataRefresh forces a metadata refresh.
func (s *Server) SourceControllerForceMetadataRefresh(w http.ResponseWriter, r *http.Request) {
	s.sourceForceAction(w, r, "Metadata refresh enqueued.", func(ctx context.Context, source *store.Source) error {
		_, err := s.App.SourceMetadataStorageWorkerKickoffWithTask(ctx, source)
		return err
	})
}

// SourceControllerSyncFilesOnDisk forces a file sync.
func (s *Server) SourceControllerSyncFilesOnDisk(w http.ResponseWriter, r *http.Request) {
	s.sourceForceAction(w, r, "File sync enqueued.", func(ctx context.Context, source *store.Source) error {
		_, err := s.App.FileSyncingWorkerKickoffWithTask(ctx, source)
		return err
	})
}
