package web

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
)

// MediaProfileWithCount is a media profile with its source count.
type MediaProfileWithCount struct {
	Profile     *core.MediaProfile
	SourceCount int64
}

// MediaProfileControllerIndex lists all media profiles
func (s *Server) MediaProfileControllerIndex(w http.ResponseWriter, r *http.Request) {
	ctx := ctxOf(r)

	// Get all media profiles that are not marked for deletion, ordered by name
	q := core.From[core.MediaProfile]("").
		Where(sq.Eq{"marked_for_deletion_at": nil}).
		OrderBy("name ASC")

	profiles, err := core.All[core.MediaProfile](ctx, s.App.Q(ctx), q)
	if err != nil {
		slog.Error("query media profiles failed", "err", err)
		s.Render(w, r, http.StatusInternalServerError, LayoutNone, ErrorHTML500())
		return
	}

	// Build profiles with source counts
	profilesWithCounts := make([]MediaProfileWithCount, 0, len(profiles))
	for _, profile := range profiles {
		// Count sources for this profile
		count, err := core.Scalar[int64](ctx, s.App.Q(ctx),
			sq.Select("COUNT(id)").From("sources").Where(sq.Eq{"media_profile_id": profile.ID}))
		if err != nil {
			count = 0
		}

		profilesWithCounts = append(profilesWithCounts, MediaProfileWithCount{
			Profile:     profile,
			SourceCount: count,
		})
	}

	s.Render(w, r, http.StatusOK, LayoutApp, MediaProfilesHTMLIndex(profilesWithCounts))
}

// MediaProfileControllerNew shows the form for creating a new media profile
func (s *Server) MediaProfileControllerNew(w http.ResponseWriter, r *http.Request) {
	ctx := ctxOf(r)
	templateID := r.URL.Query().Get("template_id")

	// Preload an existing media profile for faster creation
	var csStruct *core.MediaProfile
	if templateID != "" {
		id, err := strconv.ParseInt(templateID, 10, 64)
		if err == nil {
			profile, err := s.App.ProfilesGetMediaProfile(ctx, id)
			if err == nil && profile != nil {
				csStruct = profile
			}
		}
	}

	if csStruct == nil {
		csStruct = &core.MediaProfile{}
	}

	// Clone the struct and clear sensitive fields
	profileForForm := *csStruct
	profileForForm.ID = 0
	profileForForm.Name = ""
	profileForForm.MarkedForDeletionAt = nil

	changeset := s.App.ProfilesChangeMediaProfile(ctx, &profileForForm, nil)

	layout := OnboardingLayout(r.Context())
	s.Render(w, r, http.StatusOK, layout, MediaProfilesHTMLNew(changeset))
}

// MediaProfileControllerCreate creates a new media profile
func (s *Server) MediaProfileControllerCreate(w http.ResponseWriter, r *http.Request) {
	ctx := ctxOf(r)
	params := ParseForm(r, "media_profile")

	profile, err := s.App.ProfilesCreateMediaProfile(ctx, params)
	if err == nil {
		onboarding, _ := s.App.SettingsGet(ctx, "onboarding")
		if onboarding, ok := onboarding.(bool); ok && onboarding {
			s.PutFlash(w, r, "info", "Media profile created successfully.")
			s.Redirect(w, r, P(ctx, "/?onboarding=1"))
		} else {
			s.PutFlash(w, r, "info", "Media profile created successfully.")
			s.Redirect(w, r, P(ctx, "/media_profiles/%v", profile.ID))
		}
		return
	}

	// Changeset error
	cs, ok := core.AsChangesetError(err)
	if ok {
		layout := OnboardingLayout(r.Context())
		s.Render(w, r, http.StatusOK, layout, MediaProfilesHTMLNew(cs))
		return
	}

	slog.Error("create media profile failed", "err", err)
	s.Render(w, r, http.StatusInternalServerError, LayoutNone, ErrorHTML500())
}

// MediaProfileControllerShow displays a media profile
func (s *Server) MediaProfileControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := ctxOf(r)
	id := URLParam(r, "id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		s.Fail(w, r, core.ErrNotFound)
		return
	}

	profile, err := s.App.ProfilesGetMediaProfile(ctx, idInt)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Get sources for this profile
	q := core.SourcesQueryNew().Where(core.SourcesQueryForMediaProfile(profile.ID)).
		OrderBy("custom_name ASC")
	sources, err := core.All[core.Source](ctx, s.App.Q(ctx), q)
	if err != nil {
		slog.Error("query sources failed", "err", err)
		s.Render(w, r, http.StatusInternalServerError, LayoutNone, ErrorHTML500())
		return
	}

	s.Render(w, r, http.StatusOK, LayoutApp, MediaProfilesHTMLShow(profile, sources))
}

// MediaProfileControllerEdit shows the form for editing a media profile
func (s *Server) MediaProfileControllerEdit(w http.ResponseWriter, r *http.Request) {
	ctx := ctxOf(r)
	id := URLParam(r, "id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		s.Fail(w, r, core.ErrNotFound)
		return
	}

	profile, err := s.App.ProfilesGetMediaProfile(ctx, idInt)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	changeset := s.App.ProfilesChangeMediaProfile(ctx, profile, nil)
	s.Render(w, r, http.StatusOK, LayoutApp, MediaProfilesHTMLEdit(profile, changeset))
}

// MediaProfileControllerUpdate updates a media profile
func (s *Server) MediaProfileControllerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := ctxOf(r)
	id := URLParam(r, "id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		s.Fail(w, r, core.ErrNotFound)
		return
	}

	profile, err := s.App.ProfilesGetMediaProfile(ctx, idInt)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	params := ParseForm(r, "media_profile")
	updated, err := s.App.ProfilesUpdateMediaProfile(ctx, profile, params)
	if err == nil {
		s.PutFlash(w, r, "info", "Media profile updated successfully.")
		s.Redirect(w, r, P(ctx, "/media_profiles/%v", updated.ID))
		return
	}

	// Changeset error
	cs, ok := core.AsChangesetError(err)
	if ok {
		s.Render(w, r, http.StatusOK, LayoutApp, MediaProfilesHTMLEdit(profile, cs))
		return
	}

	slog.Error("update media profile failed", "err", err)
	s.Render(w, r, http.StatusInternalServerError, LayoutNone, ErrorHTML500())
}

// MediaProfileControllerDelete marks a media profile for deletion
func (s *Server) MediaProfileControllerDelete(w http.ResponseWriter, r *http.Request) {
	ctx := ctxOf(r)
	id := URLParam(r, "id")
	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		s.Fail(w, r, core.ErrNotFound)
		return
	}

	profile, err := s.App.ProfilesGetMediaProfile(ctx, idInt)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Parse delete_files parameter
	deleteFiles := r.URL.Query().Get("delete_files") == "true"

	// Update profile with marked_for_deletion_at timestamp
	_, err = s.App.ProfilesUpdateMediaProfile(ctx, profile, core.Attrs{
		"marked_for_deletion_at": time.Now().UTC(),
	})
	if err != nil {
		slog.Error("mark profile for deletion failed", "err", err)
		s.Render(w, r, http.StatusInternalServerError, LayoutNone, ErrorHTML500())
		return
	}

	// Enqueue deletion job
	s.App.MediaProfileDeletionWorkerKickoff(ctx, profile, core.Attrs{"delete_files": deleteFiles}, nil)

	s.PutFlash(w, r, "info", "Media Profile deletion started. This may take a while to complete.")
	s.Redirect(w, r, P(ctx, "/media_profiles"))
}
