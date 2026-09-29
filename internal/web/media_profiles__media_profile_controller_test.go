package web_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestMediaProfileController_Index(t *testing.T) {
	t.Run("lists all media_profiles", func(t *testing.T) {
		c := webtest.New(t)

		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})

		res := c.Get("/media_profiles")
		html := res.HTML(t, http.StatusOK)

		if !strings.Contains(html, "Media Profiles") {
			t.Errorf("expected 'Media Profiles' in response")
		}
		if !strings.Contains(html, profile.Name) {
			t.Errorf("expected profile name '%s' in response", profile.Name)
		}
	})

	t.Run("omits profiles that have marked_for_deletion_at set", func(t *testing.T) {
		c := webtest.New(t)

		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{
			MarkedForDeletionAt: store.Ptr(time.Now().UTC()),
		})

		res := c.Get("/media_profiles")
		html := res.HTML(t, http.StatusOK)

		if strings.Contains(html, profile.Name) {
			t.Errorf("expected profile name not to be in response")
		}
	})
}

func TestMediaProfileController_New(t *testing.T) {
	t.Run("renders form", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/media_profiles/new")
		html := res.HTML(t, http.StatusOK)

		if !strings.Contains(html, "New Media Profile") {
			t.Errorf("expected 'New Media Profile' in response")
		}
	})

	t.Run("renders correct layout when onboarding", func(t *testing.T) {
		c := webtest.New(t)
		c.App.SetSetting(c.Ctx, "onboarding", true)

		res := c.Get("/media_profiles/new")
		html := res.HTML(t, http.StatusOK)

		if strings.Contains(html, "Main navigation") {
			t.Errorf("expected 'Main navigation' NOT to be in onboarding response")
		}
	})

	t.Run("preloads some attributes when using a template", func(t *testing.T) {
		c := webtest.New(t)
		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{
			Name:     store.Ptr("My first profile"),
			SubLangs: store.Ptr("de"),
		})

		res := c.Get("/media_profiles/new", map[string]string{
			"template_id": fmt.Sprint(profile.ID),
		})
		html := res.HTML(t, http.StatusOK)

		if !strings.Contains(html, "New Media Profile") {
			t.Errorf("expected 'New Media Profile' in response")
		}
		if !strings.Contains(html, profile.SubLangs) {
			t.Errorf("expected profile sub_langs in response")
		}
		if strings.Contains(html, profile.Name) {
			t.Errorf("expected profile name NOT to be in response")
		}
	})
}

func TestMediaProfileController_Create(t *testing.T) {
	t.Run("redirects to show when data is valid", func(t *testing.T) {
		c := webtest.New(t)
		c.App.SetSetting(c.Ctx, "onboarding", false)

		attrs := map[string]any{
			"name":                 "test profile",
			"output_path_template": "output.{{ ext }}",
		}

		res := c.Post("/media_profiles", "media_profile", attrs)

		profiles, err := c.App.ListMediaProfiles(c.Ctx)
		if err != nil || len(profiles) != 1 {
			t.Fatalf("expected exactly one created media profile, got %v (err=%v)", profiles, err)
		}
		id := profiles[0].ID

		redirectTo := res.RedirectedTo(t)
		expected := "/media_profiles/" + fmt.Sprint(id)
		if redirectTo != expected {
			t.Errorf("expected redirect to %s, got %s", expected, redirectTo)
		}

		res2 := c.Get(fmt.Sprintf("/media_profiles/%d", id))
		html := res2.HTML(t, http.StatusOK)
		if !strings.Contains(html, "Media Profile") {
			t.Errorf("expected 'Media Profile' in response")
		}
	})

	t.Run("renders errors when data is invalid", func(t *testing.T) {
		c := webtest.New(t)

		attrs := map[string]any{
			"name":                 nil,
			"output_path_template": nil,
		}

		res := c.Post("/media_profiles", "media_profile", attrs)
		html := res.HTML(t, http.StatusOK)

		if !strings.Contains(html, "New Media Profile") {
			t.Errorf("expected 'New Media Profile' in error response")
		}
	})

	t.Run("redirects to onboarding when onboarding", func(t *testing.T) {
		c := webtest.New(t)
		c.App.SetSetting(c.Ctx, "onboarding", true)

		attrs := map[string]any{
			"name":                 "test profile",
			"output_path_template": "output.{{ ext }}",
		}

		res := c.Post("/media_profiles", "media_profile", attrs)

		redirectTo := res.RedirectedTo(t)
		if redirectTo != "/?onboarding=1" {
			t.Errorf("expected redirect to /?onboarding=1, got %s", redirectTo)
		}
	})

	t.Run("renders correct layout on error when onboarding", func(t *testing.T) {
		c := webtest.New(t)
		c.App.SetSetting(c.Ctx, "onboarding", true)

		attrs := map[string]any{
			"name":                 nil,
			"output_path_template": nil,
		}

		res := c.Post("/media_profiles", "media_profile", attrs)

		html := res.HTML(t, http.StatusOK)
		if strings.Contains(html, "Main navigation") {
			t.Errorf("expected 'Main navigation' NOT to be in onboarding error response")
		}
	})
}

func TestMediaProfileController_Edit(t *testing.T) {
	t.Run("renders form for editing chosen media_profile", func(t *testing.T) {
		c := webtest.New(t)

		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})

		res := c.Get("/media_profiles/" + fmt.Sprint(profile.ID) + "/edit")
		html := res.HTML(t, http.StatusOK)

		if !strings.Contains(html, profile.Name) {
			t.Errorf("expected profile name '%s' in response", profile.Name)
		}
	})
}

func TestMediaProfileController_Update(t *testing.T) {
	t.Run("redirects when data is valid", func(t *testing.T) {
		c := webtest.New(t)

		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})

		attrs := map[string]any{
			"name":                 "updated name",
			"output_path_template": "new_template.{{ ext }}",
		}

		res := c.Patch("/media_profiles/"+fmt.Sprint(profile.ID), "media_profile", attrs)
		redirectTo := res.RedirectedTo(t)

		expectedPath := "/media_profiles/" + fmt.Sprint(profile.ID)
		if redirectTo != expectedPath {
			t.Errorf("expected redirect to %s, got %s", expectedPath, redirectTo)
		}
	})

	t.Run("renders errors when data is invalid", func(t *testing.T) {
		c := webtest.New(t)

		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})

		attrs := map[string]any{
			"name":                 nil,
			"output_path_template": nil,
		}

		res := c.Patch("/media_profiles/"+fmt.Sprint(profile.ID), "media_profile", attrs)
		html := res.HTML(t, http.StatusOK)

		if !strings.Contains(html, profile.Name) {
			t.Errorf("expected profile name in error response")
		}
	})
}

func TestMediaProfileController_Delete(t *testing.T) {
	t.Run("redirects to the media_profiles page", func(t *testing.T) {
		c := webtest.New(t)

		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})

		res := c.Delete("/media_profiles/" + fmt.Sprint(profile.ID))
		redirectTo := res.RedirectedTo(t)

		if redirectTo != "/media_profiles" {
			t.Errorf("expected redirect to /media_profiles, got %s", redirectTo)
		}
	})

	t.Run("sets marked_for_deletion_at", func(t *testing.T) {
		c := webtest.New(t)

		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})

		c.Delete("/media_profiles/" + fmt.Sprint(profile.ID))

		updated, err := c.App.GetMediaProfile(c.Ctx, profile.ID)
		if err != nil {
			t.Fatalf("failed to reload profile: %v", err)
		}

		if updated.MarkedForDeletionAt == nil {
			t.Errorf("expected marked_for_deletion_at to be set")
		}
	})

	t.Run("enqueues a job without the delete_files arg", func(t *testing.T) {
		c := webtest.New(t)

		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})

		c.Delete("/media_profiles/" + fmt.Sprint(profile.ID))

		job := c.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: app.MediaProfileDeletionWorkerName,
			Args:   map[string]any{"delete_files": false},
		})

		if job == nil {
			t.Error("expected enqueued job not found")
		}
	})

	t.Run("enqueues a job with the delete_files arg", func(t *testing.T) {
		c := webtest.New(t)

		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})

		c.Delete("/media_profiles/" + fmt.Sprint(profile.ID) + "?delete_files=true")

		job := c.Oban.AssertEnqueued(t, obanlite.Match{
			Worker: app.MediaProfileDeletionWorkerName,
			Args:   map[string]any{"delete_files": true},
		})

		if job == nil {
			t.Error("expected enqueued job not found")
		}
	})
}
