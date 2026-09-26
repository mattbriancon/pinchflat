package core_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/db"
)

func TestProfiles_Schema(t *testing.T) {
	t.Run("can be JSON encoded without error", func(t *testing.T) {
		ta := coretest.NewApp(t)
		profile := coretest.MediaProfileFixture(t, ta, core.Attrs{})

		_, err := db.EncodeJSON(profile)
		if err != nil {
			t.Errorf("EncodeJSON failed: %v", err)
		}
	})
}

func TestProfiles_ListMediaProfiles(t *testing.T) {
	t.Run("it returns all media_profiles", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})

		profiles, err := ta.ProfilesListMediaProfiles(ta.Ctx)
		if err != nil {
			t.Fatalf("ProfilesListMediaProfiles failed: %v", err)
		}

		if len(profiles) != 1 {
			t.Errorf("expected 1 profile, got %d", len(profiles))
		}
		if profiles[0].ID != mediaProfile.ID {
			t.Errorf("expected profile ID %d, got %d", mediaProfile.ID, profiles[0].ID)
		}
	})
}

func TestProfiles_GetMediaProfile(t *testing.T) {
	t.Run("it returns the media_profile with given id", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})

		retrieved, err := ta.ProfilesGetMediaProfile(ta.Ctx, mediaProfile.ID)
		if err != nil {
			t.Fatalf("ProfilesGetMediaProfile failed: %v", err)
		}

		if retrieved.ID != mediaProfile.ID {
			t.Errorf("expected profile ID %d, got %d", mediaProfile.ID, retrieved.ID)
		}
		if retrieved.Name != mediaProfile.Name {
			t.Errorf("expected name %s, got %s", mediaProfile.Name, retrieved.Name)
		}
	})
}

func TestProfiles_CreateMediaProfile(t *testing.T) {
	t.Run("creation with valid data creates a media_profile", func(t *testing.T) {
		ta := coretest.NewApp(t)
		validAttrs := core.Attrs{
			"name":                 "some name",
			"output_path_template": "output_template.{{ ext }}",
		}

		profile, err := ta.ProfilesCreateMediaProfile(ta.Ctx, validAttrs)
		if err != nil {
			t.Fatalf("ProfilesCreateMediaProfile failed: %v", err)
		}

		if profile.Name != "some name" {
			t.Errorf("expected name 'some name', got %s", profile.Name)
		}
		if profile.OutputPathTemplate != "output_template.{{ ext }}" {
			t.Errorf("expected output_path_template 'output_template.{{ ext }}', got %s", profile.OutputPathTemplate)
		}
	})

	t.Run("creation with invalid data returns error changeset", func(t *testing.T) {
		ta := coretest.NewApp(t)
		invalidAttrs := core.Attrs{
			"name":                 nil,
			"output_path_template": nil,
		}

		_, err := ta.ProfilesCreateMediaProfile(ta.Ctx, invalidAttrs)
		if err == nil {
			t.Error("expected error for invalid attrs, got nil")
		}

		csErr, ok := core.AsChangesetError(err)
		if !ok || csErr == nil {
			t.Errorf("expected ChangesetError, got %T", err)
		}
	})
}

func TestProfiles_UpdateMediaProfile(t *testing.T) {
	t.Run("updating with valid data updates the media_profile", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})

		updateAttrs := core.Attrs{
			"name":                 "some updated name",
			"output_path_template": "new_output_template.{{ ext }}",
		}

		updated, err := ta.ProfilesUpdateMediaProfile(ta.Ctx, mediaProfile, updateAttrs)
		if err != nil {
			t.Fatalf("ProfilesUpdateMediaProfile failed: %v", err)
		}

		if updated.Name != "some updated name" {
			t.Errorf("expected name 'some updated name', got %s", updated.Name)
		}
		if updated.OutputPathTemplate != "new_output_template.{{ ext }}" {
			t.Errorf("expected output_path_template 'new_output_template.{{ ext }}', got %s", updated.OutputPathTemplate)
		}
	})

	t.Run("updating with invalid data returns error changeset", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})

		invalidAttrs := core.Attrs{
			"name":                 nil,
			"output_path_template": nil,
		}

		_, err := ta.ProfilesUpdateMediaProfile(ta.Ctx, mediaProfile, invalidAttrs)
		if err == nil {
			t.Error("expected error for invalid attrs, got nil")
		}

		// Verify the original profile hasn't changed
		retrieved, err := ta.ProfilesGetMediaProfile(ta.Ctx, mediaProfile.ID)
		if err != nil {
			t.Fatalf("ProfilesGetMediaProfile failed: %v", err)
		}

		if retrieved.ID != mediaProfile.ID {
			t.Errorf("expected profile ID to match original")
		}
	})
}

func TestProfiles_DeleteMediaProfile(t *testing.T) {
	t.Run("deletion deletes the media_profile", func(t *testing.T) {
		t.Skip("BLOCKED: SourcesDeleteSource unported")
	})

	t.Run("deletion deletes all sources", func(t *testing.T) {
		t.Skip("BLOCKED: SourcesDeleteSource unported")
	})

	t.Run("deletion deletes all media items", func(t *testing.T) {
		t.Skip("BLOCKED: SourcesDeleteSource unported")
	})

	t.Run("deletion does not delete files by default", func(t *testing.T) {
		t.Skip("BLOCKED: SourcesDeleteSource unported")
	})
}

func TestProfiles_DeleteMediaProfile_WhenDeletingFiles(t *testing.T) {
	t.Run("still deletes all the needful records", func(t *testing.T) {
		t.Skip("BLOCKED: SourcesDeleteSource unported")
	})

	t.Run("deletes files", func(t *testing.T) {
		t.Skip("BLOCKED: SourcesDeleteSource unported")
	})
}

func TestProfiles_ChangeMediaProfile(t *testing.T) {
	t.Run("it returns a media_profile changeset", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaProfile := coretest.MediaProfileFixture(t, ta, core.Attrs{})

		cs := ta.ProfilesChangeMediaProfile(ta.Ctx, mediaProfile, core.Attrs{})
		if cs == nil {
			t.Error("expected changeset, got nil")
		}
	})

	t.Run("it ensures the media profile's output template ends with an extension", func(t *testing.T) {
		ta := coretest.NewApp(t)

		validTemplates := []string{
			"output_template.{{ ext }}",
			"output_template.{{ext}}",
			"output_template.%(ext)s",
			"output_template.%(ext)S",
			"output_template.%( ext )s",
			"output_template.%( ext )S",
		}

		for _, template := range validTemplates {
			cs := ta.ProfilesChangeMediaProfile(ta.Ctx, &core.MediaProfile{}, core.Attrs{
				"name":                 "a",
				"output_path_template": template,
			})

			if !cs.Valid() {
				t.Errorf("expected valid changeset for template %q, but it was invalid", template)
			}
		}
	})

	t.Run("it does not allow invalid output templates", func(t *testing.T) {
		ta := coretest.NewApp(t)

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
			cs := ta.ProfilesChangeMediaProfile(ta.Ctx, &core.MediaProfile{}, core.Attrs{
				"name":                 "a",
				"output_path_template": template,
			})

			if cs.Valid() {
				t.Errorf("expected invalid changeset for template %q, but it was valid", template)
			}
		}
	})
}
