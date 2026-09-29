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
	profile := apptest.MediaProfileFixture(t, ta, store.Attrs{})

	_, err := db.EncodeJSON(profile)
	if err != nil {
		t.Errorf("EncodeJSON failed: %v", err)
	}
}

func TestProfiles_ListMediaProfiles(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)
	mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})

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
	mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})

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
		validAttrs := store.Attrs{
			"name":                 "some name",
			"output_path_template": "output_template.{{ ext }}",
		}

		profile, err := ta.CreateMediaProfile(ta.Ctx, validAttrs)
		if err != nil {
			t.Fatalf("ProfilesCreateMediaProfile failed: %v", err)
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
		invalidAttrs := store.Attrs{
			"name":                 nil,
			"output_path_template": nil,
		}

		_, err := ta.CreateMediaProfile(ta.Ctx, invalidAttrs)
		if err == nil {
			t.Error("expected error for invalid attrs")
		}

		csErr, ok := store.AsChangesetError(err)
		if !ok || csErr == nil {
			t.Errorf("expected store.ChangesetError, got %T", err)
		}
	})
}

func TestProfiles_UpdateMediaProfile(t *testing.T) {
	t.Run("valid data", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})

		updateAttrs := store.Attrs{
			"name":                 "updated name",
			"output_path_template": "new_output_template.{{ ext }}",
		}

		updated, err := ta.UpdateMediaProfile(ta.Ctx, mediaProfile, updateAttrs)
		if err != nil {
			t.Fatalf("ProfilesUpdateMediaProfile failed: %v", err)
		}

		if updated.Name != "updated name" {
			t.Errorf("expected name 'updated name', got %s", updated.Name)
		}
		if updated.OutputPathTemplate != "new_output_template.{{ ext }}" {
			t.Errorf("expected template 'new_output_template.{{ ext }}', got %s", updated.OutputPathTemplate)
		}
	})

	t.Run("invalid data", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})

		invalidAttrs := store.Attrs{
			"name":                 nil,
			"output_path_template": nil,
		}

		_, err := ta.UpdateMediaProfile(ta.Ctx, mediaProfile, invalidAttrs)
		if err == nil {
			t.Error("expected error for invalid attrs")
		}

		retrieved, err := ta.GetMediaProfile(ta.Ctx, mediaProfile.ID)
		if err != nil {
			t.Fatalf("ProfilesGetMediaProfile failed: %v", err)
		}

		if retrieved.ID != mediaProfile.ID {
			t.Error("profile should not have changed")
		}
	})
}

func TestProfiles_DeleteMediaProfile(t *testing.T) {
	t.Run("deletes profile", func(t *testing.T) {
		t.Parallel()
		ta := apptest.NewApp(t)
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, store.KW{})
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
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})
		source := apptest.SourceFixture(t, ta, store.Attrs{
			"media_profile_id": mediaProfile.ID,
		})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, store.KW{})
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
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})
		source := apptest.SourceFixture(t, ta, store.Attrs{
			"media_profile_id": mediaProfile.ID,
		})
		mediaItem := apptest.MediaItemFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, store.KW{})
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
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})
		source := apptest.SourceFixture(t, ta, store.Attrs{
			"media_profile_id": mediaProfile.ID,
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, store.KW{})
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

		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})
		source := apptest.SourceFixture(t, ta, store.Attrs{
			"media_profile_id": mediaProfile.ID,
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, store.KW{store.Opt("delete_files", true)})
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

		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})
		source := apptest.SourceFixture(t, ta, store.Attrs{
			"media_profile_id": mediaProfile.ID,
		})
		mediaItem := apptest.MediaItemWithAttachmentsFixture(t, ta, store.MediaItemParams{
			SourceID: store.Ptr(source.ID),
		})

		mediaFilepath := *mediaItem.MediaFilepath

		_, err := ta.ProfilesDeleteMediaProfile(ta.Ctx, mediaProfile, store.KW{store.Opt("delete_files", true)})
		if err != nil {
			t.Fatalf("ProfilesDeleteMediaProfile failed: %v", err)
		}

		if _, err := os.Stat(mediaFilepath); !os.IsNotExist(err) {
			t.Error("expected media file to not exist")
		}
	})
}

func TestProfiles_ChangeMediaProfile(t *testing.T) {
	t.Parallel()
	ta := apptest.NewApp(t)

	t.Run("returns changeset", func(t *testing.T) {
		t.Parallel()
		mediaProfile := apptest.MediaProfileFixture(t, ta, store.Attrs{})
		cs := ta.ChangeMediaProfile(ta.Ctx, mediaProfile, store.Attrs{})
		if cs == nil {
			t.Error("expected changeset, got nil")
		}
	})

	t.Run("allows valid templates", func(t *testing.T) {
		t.Parallel()
		validTemplates := []string{
			"output_template.{{ ext }}",
			"output_template.{{ext}}",
			"output_template.%(ext)s",
			"output_template.%(ext)S",
			"output_template.%( ext )s",
			"output_template.%( ext )S",
		}

		for _, template := range validTemplates {
			cs := ta.ChangeMediaProfile(ta.Ctx, &store.MediaProfile{}, store.Attrs{
				"name":                 "a",
				"output_path_template": template,
			})
			if !cs.Valid() {
				t.Errorf("template %q should be valid", template)
			}
		}
	})

	t.Run("rejects invalid templates", func(t *testing.T) {
		t.Parallel()
		invalidTemplates := []string{
			"output_template.{{ ext }}.something",
			"output_template.{{   ext   }}",
			"output_template{{ ext }}",
			"output_template.%(ext)s.something",
			"output_template.txt",
			"output_template%(ext)s",
			"output_template.%(nope)s",
			"output_template",
		}

		for _, template := range invalidTemplates {
			cs := ta.ChangeMediaProfile(ta.Ctx, &store.MediaProfile{}, store.Attrs{
				"name":                 "a",
				"output_path_template": template,
			})
			if cs.Valid() {
				t.Errorf("template %q should be invalid", template)
			}
		}
	})
}
