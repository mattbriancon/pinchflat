package web

// Port of lib/pinchflat_web/controllers/sources/source_controller.ex.

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
)

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
	var cs *core.Source
	if templateIDStr != "" {
		templateID, _ := strconv.ParseInt(templateIDStr, 10, 64)
		if templateID > 0 {
			source, err := s.App.SourcesGetSource(ctx, templateID)
			if err == nil {
				cs = source
			}
		}
	}
	if cs == nil {
		cs = &core.Source{}
	}

	// Create a blank source with nullified fields from the template
	source := &core.Source{
		IndexFrequencyMinutes:      cs.IndexFrequencyMinutes,
		DownloadCutoffDate:         cs.DownloadCutoffDate,
		TitleFilterRegex:           cs.TitleFilterRegex,
		OutputPathTemplateOverride: cs.OutputPathTemplateOverride,
		CookieBehaviour:            cs.CookieBehaviour,
		MediaProfileID:             cs.MediaProfileID,
	}

	mediaProfiles, _ := s.App.ProfilesListMediaProfiles(ctx)
	changeset := s.App.SourcesChangeSource(ctx, source, core.Attrs{}, "")

	layout := OnboardingLayout(ctx)
	s.Render(w, r, http.StatusOK, layout, SourceHTMLNew(changeset, mediaProfiles))
}

// SourceControllerCreate creates a new source.
func (s *Server) SourceControllerCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sourceParams := ParseForm(r, "source")

	source, err := s.App.SourcesCreateSource(ctx, sourceParams, core.KW{})
	if err != nil {
		var csErr *core.ChangesetError
		if errors.As(err, &csErr) {
			// Re-render form with errors
			mediaProfiles, _ := s.App.ProfilesListMediaProfiles(ctx)
			layout := OnboardingLayout(ctx)
			s.Render(w, r, http.StatusOK, layout, SourceHTMLNew(csErr.Changeset, mediaProfiles))
			return
		}
		s.Fail(w, r, err)
		return
	}

	// Determine redirect location
	onboarding, _ := s.App.SettingsGet(ctx, "onboarding")
	var redirectPath string
	if onboarding == true {
		redirectPath = P(ctx, "/?onboarding=1")
	} else {
		redirectPath = P(ctx, "/sources/%v", source.ID)
	}

	s.PutFlash(w, r, "info", "Source created successfully.")
	s.Redirect(w, r, redirectPath)
}

// SourceControllerShow displays a source.
func (s *Server) SourceControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(URLParam(r, "id"), 10, 64)

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	source, _ = s.App.PreloadSourceMediaProfile(ctx, source)

	pendingTasks, _ := s.App.TasksListTasksFor(ctx, source, nil, []string{"executing", "available", "scheduled", "retryable"})
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
	id, _ := strconv.ParseInt(URLParam(r, "id"), 10, 64)

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	mediaProfiles, _ := s.App.ProfilesListMediaProfiles(ctx)
	changeset := s.App.SourcesChangeSource(ctx, source, core.Attrs{}, "")

	s.Render(w, r, http.StatusOK, LayoutApp, SourceHTMLEdit(source, changeset, mediaProfiles))
}

// SourceControllerUpdate updates a source.
func (s *Server) SourceControllerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(URLParam(r, "id"), 10, 64)
	sourceParams := ParseForm(r, "source")

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	updated, err := s.App.SourcesUpdateSource(ctx, source, sourceParams, core.KW{})
	if err != nil {
		var csErr *core.ChangesetError
		if errors.As(err, &csErr) {
			// Re-render form with errors, passing the original loaded source
			mediaProfiles, _ := s.App.ProfilesListMediaProfiles(ctx)
			s.Render(w, r, http.StatusOK, LayoutApp, SourceHTMLEdit(source, csErr.Changeset, mediaProfiles))
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
	id, _ := strconv.ParseInt(URLParam(r, "id"), 10, 64)
	deleteFiles := r.URL.Query().Get("delete_files") == "true"

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Mark for deletion
	_, err = s.App.SourcesUpdateSource(ctx, source, core.Attrs{
		"marked_for_deletion_at": time.Now().UTC(),
	}, core.KW{})
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Kickoff deletion worker
	_, _ = s.App.SourceDeletionWorkerKickoff(ctx, source, core.Attrs{
		"delete_files": deleteFiles,
	}, core.KW{})

	s.PutFlash(w, r, "info", "Source deletion started. This may take a while to complete.")
	s.Redirect(w, r, P(ctx, "/sources"))
}

// SourceControllerForceDownloadPending forces pending media downloads.
func (s *Server) SourceControllerForceDownloadPending(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(URLParam(r, "source_id"), 10, 64)

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	if err := s.App.DownloadingHelpersEnqueuePendingDownloadTasks(ctx, source, core.KW{}); err != nil {
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "Forcing download of pending media items.")
	s.Redirect(w, r, P(ctx, "/sources/%v", source.ID))
}

// SourceControllerForceRedownload forces redownload of existing media.
func (s *Server) SourceControllerForceRedownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(URLParam(r, "source_id"), 10, 64)

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	if _, err := s.App.DownloadingHelpersKickoffRedownloadForExistingMedia(ctx, source); err != nil {
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "Forcing re-download of downloaded media items.")
	s.Redirect(w, r, P(ctx, "/sources/%v", source.ID))
}

// SourceControllerForceIndex forces an indexing task.
func (s *Server) SourceControllerForceIndex(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(URLParam(r, "source_id"), 10, 64)

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	if _, err := s.App.SlowIndexingHelpersKickoffIndexingTask(ctx, source, core.Attrs{"force": true}, core.KW{}); err != nil {
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "Index enqueued.")
	s.Redirect(w, r, P(ctx, "/sources/%v", source.ID))
}

// SourceControllerForceMetadataRefresh forces a metadata refresh.
func (s *Server) SourceControllerForceMetadataRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(URLParam(r, "source_id"), 10, 64)

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	if _, err := s.App.SourceMetadataStorageWorkerKickoffWithTask(ctx, source, core.KW{}); err != nil {
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "Metadata refresh enqueued.")
	s.Redirect(w, r, P(ctx, "/sources/%v", source.ID))
}

// SourceControllerSyncFilesOnDisk forces a file sync.
func (s *Server) SourceControllerSyncFilesOnDisk(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, _ := strconv.ParseInt(URLParam(r, "source_id"), 10, 64)

	source, err := s.App.SourcesGetSource(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	if _, err := s.App.FileSyncingWorkerKickoffWithTask(ctx, source, core.KW{}); err != nil {
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "File sync enqueued.")
	s.Redirect(w, r, P(ctx, "/sources/%v", source.ID))
}
