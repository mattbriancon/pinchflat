package app_test

import (
	"os"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func TestProfiles_Schema(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)
	profile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})

	_, err := db.EncodeJSON(profile)
	if err != nil {
		t.Errorf("EncodeJSON failed: %v", err)
	}
}

func TestProfiles_ListMediaProfiles(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)
	mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})

	profiles, err := ta.ListMediaProfiles(ta.Ctx)
	if err != nil {
		t.Fatalf("ProfilesListMediaProfiles failed: %v", err)
	}

	if len(profiles) != 1 {
		t.Errorf("expected 1 profile, got %d", len(profiles))
	}
	if profiles[0].ID != mediaProfile.ID {
		t.Errorf("expected profile ID %d, got %d", mediaProfile.ID, profiles[0].ID)
	}
}

func TestProfiles_GetMediaProfile(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)
	mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})

	retrieved, err := ta.GetMediaProfile(ta.Ctx, mediaProfile.ID)
	if err != nil {
		t.Fatalf("ProfilesGetMediaProfile failed: %v", err)
	}

	if retrieved.ID != mediaProfile.ID {
		t.Errorf("expected profile ID %d, got %d", mediaProfile.ID, retrieved.ID)
	}
	if retrieved.Name != mediaProfile.Name {
		t.Errorf("expected name %s, got %s", mediaProfile.Name, retrieved.Name)
	}
}

func TestProfiles_CreateMediaProfile(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		valid := store.MediaProfileParams{
			Name:               store.Ptr("some name"),
			OutputPathTemplate: store.Ptr("output_template.{{ ext }}"),
		}

		profile, errs, err := ta.CreateMediaProfile(ta.Ctx, valid)
		if err != nil || len(errs) > 0 {
			t.Fatalf("CreateMediaProfile failed: %v %v", errs, err)
		}

		if profile.Name != "some name" {
			t.Errorf("expected name 'some name', got %s", profile.Name)
		}
		if profile.OutputPathTemplate != "output_template.{{ ext }}" {
			t.Errorf("expected template 'output_template.{{ ext }}', got %s", profile.OutputPathTemplate)
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)

		profile, errs, err := ta.CreateMediaProfile(ta.Ctx, store.MediaProfileParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if profile != nil || len(errs) == 0 {
			t.Errorf("expected validation errors, got profile=%v errs=%v", profile, errs)
		}
	})

	t.Run("duplicate name", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		existing := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})

		_, errs, err := ta.CreateMediaProfile(ta.Ctx, store.MediaProfileParams{
			Name:               store.Ptr(existing.Name),
			OutputPathTemplate: store.Ptr("x.{{ ext }}"),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := errs["name"]; len(got) != 1 || got[0] != "has already been taken" {
			t.Errorf("expected name taken error, got %v", errs)
		}
	})
}

func TestProfiles_UpdateMediaProfile(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})

		updated, errs, err := ta.UpdateMediaProfile(ta.Ctx, mediaProfile, store.MediaProfileParams{
			Name:               store.Ptr("updated name"),
			OutputPathTemplate: store.Ptr("new_output_template.{{ ext }}"),
		})
		if err != nil || len(errs) > 0 {
			t.Fatalf("UpdateMediaProfile failed: %v %v", errs, err)
		}

		if updated.Name != "updated name" {
			t.Errorf("expected name 'updated name', got %s", updated.Name)
		}
		if updated.OutputPathTemplate != "new_output_template.{{ ext }}" {
			t.Errorf("expected template 'new_output_template.{{ ext }}', got %s", updated.OutputPathTemplate)
		}
		reloaded, err := ta.GetMediaProfile(ta.Ctx, mediaProfile.ID)
		if err != nil {
			t.Fatalf("GetMediaProfile failed: %v", err)
		}
		if reloaded.Name != "updated name" {
			t.Errorf("expected persisted name 'updated name', got %s", reloaded.Name)
		}
	})

	t.Run("only writes submitted fields", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{
			RedownloadDelayDays: store.Ptr(3),
			AudioTrack:          store.Ptr("de"),
		})

		if _, errs, err := ta.UpdateMediaProfile(ta.Ctx, mediaProfile, store.MediaProfileParams{SubLangs: store.Ptr("fr")}); err != nil || len(errs) > 0 {
			t.Fatalf("UpdateMediaProfile failed: %v %v", errs, err)
		}
		reloaded, _ := ta.GetMediaProfile(ta.Ctx, mediaProfile.ID)
		if reloaded.SubLangs != "fr" || reloaded.RedownloadDelayDays == nil || *reloaded.RedownloadDelayDays != 3 || reloaded.AudioTrack == nil {
			t.Errorf("unexpected profile after partial update: %+v", reloaded)
		}
	})

	t.Run("blank clears nullable fields", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{
			RedownloadDelayDays: store.Ptr(3),
			AudioTrack:          store.Ptr("de"),
		})

		_, errs, err := ta.UpdateMediaProfile(ta.Ctx, mediaProfile, store.MediaProfileParams{
			AudioTrack:               store.Ptr(""),
			ClearRedownloadDelayDays: true,
		})
		if err != nil || len(errs) > 0 {
			t.Fatalf("UpdateMediaProfile failed: %v %v", errs, err)
		}
		reloaded, _ := ta.GetMediaProfile(ta.Ctx, mediaProfile.ID)
		if reloaded.AudioTrack != nil || reloaded.RedownloadDelayDays != nil {
			t.Errorf("expected cleared fields, got %+v", reloaded)
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})

		_, errs, err := ta.UpdateMediaProfile(ta.Ctx, mediaProfile, store.MediaProfileParams{
			Name:               store.Ptr(""),
			OutputPathTemplate: store.Ptr(""),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(errs["name"]) == 0 || len(errs["output_path_template"]) == 0 {
			t.Errorf("expected validation errors, got %v", errs)
		}

		retrieved, err := ta.GetMediaProfile(ta.Ctx, mediaProfile.ID)
		if err != nil {
			t.Fatalf("GetMediaProfile failed: %v", err)
		}
		if retrieved.Name != mediaProfile.Name {
			t.Error("profile should not have changed")
		}
	})
}

func TestProfiles_DeleteMediaProfile(t *testing.T) {
	t.Run("deletes profile", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, false)
		if err != nil {
			t.Fatalf("ProfilesDeleteMediaProfile failed: %v", err)
		}

		_, err = ta.GetMediaProfile(ta.Ctx, mediaProfile.ID)
		if err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound for deleted profile")
		}
	})

	t.Run("deletes sources", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			MediaProfileID: store.Ptr(mediaProfile.ID),
		})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, false)
		if err != nil {
			t.Fatalf("ProfilesDeleteMediaProfile failed: %v", err)
		}

		_, err = ta.GetSource(ta.Ctx, source.ID)
		if err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound for deleted source")
		}
	})

	t.Run("deletes media items", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			MediaProfileID: store.Ptr(mediaProfile.ID),
		})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, false)
		if err != nil {
			t.Fatalf("ProfilesDeleteMediaProfile failed: %v", err)
		}

		_, err = ta.GetMediaItem(ta.Ctx, mediaItem.ID)
		if err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound for deleted media_item")
		}
	})

	t.Run("preserves files by default", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			MediaProfileID: store.Ptr(mediaProfile.ID),
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, false)
		if err != nil {
			t.Fatalf("ProfilesDeleteMediaProfile failed: %v", err)
		}

		if _, err := os.Stat(*mediaItem.MediaFilepath); os.IsNotExist(err) {
			t.Error("expected media file to exist")
		}
	})
}

func TestProfiles_DeleteMediaProfile_WhenDeletingFiles(t *testing.T) {
	t.Run("deletes records", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			MediaProfileID: store.Ptr(mediaProfile.ID),
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, true)
		if err != nil {
			t.Fatalf("ProfilesDeleteMediaProfile failed: %v", err)
		}

		_, err = ta.GetMediaProfile(ta.Ctx, mediaProfile.ID)
		if err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound for deleted profile")
		}

		_, err = ta.GetSource(ta.Ctx, source.ID)
		if err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound for deleted source")
		}

		_, err = ta.GetMediaItem(ta.Ctx, mediaItem.ID)
		if err != store.ErrNotFound {
			t.Errorf("expected store.ErrNotFound for deleted media_item")
		}
	})

	t.Run("deletes files", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		ta.UserScriptMock.Run.Stub(func(event string, data any) error {
			return nil
		})

		mediaProfile := apptest.MediaProfileFixture(t, ta, store.MediaProfileParams{})
		source := apptest.SourceFixture(t, ta, store.SourceParams{
			MediaProfileID: store.Ptr(mediaProfile.ID),
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		mediaFilepath := *mediaItem.MediaFilepath

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, true)
		if err != nil {
			t.Fatalf("ProfilesDeleteMediaProfile failed: %v", err)
		}

		if _, err := os.Stat(mediaFilepath); !os.IsNotExist(err) {
			t.Error("expected media file to not exist")
		}
	})
}

func TestProfiles_OutputPathTemplateValidation(t *testing.T) {
	t.Parallel()
	templates := []struct {
		template string
		valid    bool
	}{
		{"output_template.{{ ext }}", true},
		{"output_template.{{ext}}", true},
		{"output_template.%(ext)s", true},
		{"output_template.%(ext)S", true},
		{"output_template.%( ext )s", true},
		{"output_template.%( ext )S", true},
		{"output_template.{{ ext }}.something", false},
		{"output_template.{{   ext   }}", false},
		{"output_template{{ ext }}", false},
		{"output_template.%(ext)s.something", false},
		{"output_template.txt", false},
		{"output_template%(ext)s", false},
		{"output_template.%(nope)s", false},
		{"output_template", false},
	}
	for _, tt := range templates {
		errs := store.MediaProfileParams{
			Name:               store.Ptr("a"),
			OutputPathTemplate: store.Ptr(tt.template),
		}.Validate(&store.MediaProfile{})
		if valid := len(errs) == 0; valid != tt.valid {
			t.Errorf("template %q: valid = %v, want %v (%v)", tt.template, valid, tt.valid, errs)
		}
	}
}
