package web

import (
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
	ctx := r.Context()

	// Get all media profiles that are not marked for deletion, ordered by name
	q := core.From[core.MediaProfile]("").
		Where(sq.Eq{"marked_for_deletion_at": nil}).
		OrderBy("name ASC")

	profiles, err := core.All[core.MediaProfile](ctx, s.App.Q(ctx), q)
	if err != nil {
		s.Fail(w, r, err)
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
	ctx := r.Context()
	templateID := r.URL.Query().Get("template_id")

	// Preload an existing media profile for faster creation
	var csStruct *core.MediaProfile
	if id, err := strconv.ParseInt(templateID, 10, 64); err == nil {
		if profile, err := s.App.ProfilesGetMediaProfile(ctx, id); err == nil && profile != nil {
			csStruct = profile
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

	layout := OnboardingLayout(ctx)
	s.Render(w, r, http.StatusOK, layout, MediaProfilesHTMLNew(changeset))
}

// MediaProfileControllerCreate creates a new media profile
func (s *Server) MediaProfileControllerCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := ParseForm(r, "media_profile")

	profile, err := s.App.ProfilesCreateMediaProfile(ctx, params)
	if err != nil {
		if cs, ok := core.AsChangesetError(err); ok {
			layout := OnboardingLayout(ctx)
			s.Render(w, r, http.StatusOK, layout, MediaProfilesHTMLNew(cs))
			return
		}
		s.Fail(w, r, err)
		return
	}

	redirectPath := P(ctx, "/media_profiles/%v", profile.ID)
	if onboarding, _ := s.App.SettingsGet(ctx, "onboarding"); onboarding == true {
		redirectPath = P(ctx, "/?onboarding=1")
	}
	s.PutFlash(w, r, "info", "Media profile created successfully.")
	s.Redirect(w, r, redirectPath)
}

// MediaProfileControllerShow displays a media profile
func (s *Server) MediaProfileControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	profile, ok := loadOrFail(s, w, r, "id", s.App.ProfilesGetMediaProfile)
	if !ok {
		return
	}

	// Get sources for this profile
	q := core.SourcesQueryNew().Where(core.SourcesQueryForMediaProfile(profile.ID)).
		OrderBy("custom_name ASC")
	sources, err := core.All[core.Source](ctx, s.App.Q(ctx), q)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	s.Render(w, r, http.StatusOK, LayoutApp, MediaProfilesHTMLShow(profile, sources))
}

// MediaProfileControllerEdit shows the form for editing a media profile
func (s *Server) MediaProfileControllerEdit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	profile, ok := loadOrFail(s, w, r, "id", s.App.ProfilesGetMediaProfile)
	if !ok {
		return
	}

	changeset := s.App.ProfilesChangeMediaProfile(ctx, profile, nil)
	s.Render(w, r, http.StatusOK, LayoutApp, MediaProfilesHTMLEdit(profile, changeset))
}

// MediaProfileControllerUpdate updates a media profile
func (s *Server) MediaProfileControllerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	profile, ok := loadOrFail(s, w, r, "id", s.App.ProfilesGetMediaProfile)
	if !ok {
		return
	}

	params := ParseForm(r, "media_profile")
	updated, err := s.App.ProfilesUpdateMediaProfile(ctx, profile, params)
	if err != nil {
		if cs, ok := core.AsChangesetError(err); ok {
			s.Render(w, r, http.StatusOK, LayoutApp, MediaProfilesHTMLEdit(profile, cs))
			return
		}
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "Media profile updated successfully.")
	s.Redirect(w, r, P(ctx, "/media_profiles/%v", updated.ID))
}

// MediaProfileControllerDelete marks a media profile for deletion
func (s *Server) MediaProfileControllerDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	profile, ok := loadOrFail(s, w, r, "id", s.App.ProfilesGetMediaProfile)
	if !ok {
		return
	}
	deleteFiles := r.URL.Query().Get("delete_files") == "true"

	_, err := s.App.ProfilesUpdateMediaProfile(ctx, profile, core.Attrs{
		"marked_for_deletion_at": time.Now().UTC(),
	})
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	s.App.MediaProfileDeletionWorkerKickoff(ctx, profile, core.Attrs{"delete_files": deleteFiles}, nil)

	s.PutFlash(w, r, "info", "Media Profile deletion started. This may take a while to complete.")
	s.Redirect(w, r, P(ctx, "/media_profiles"))
}
