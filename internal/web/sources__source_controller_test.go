package web_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestSourceController_Index(t *testing.T) {
	t.Run("returns 200", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/sources")

		if !strings.Contains(res.HTML(t, 200), "Sources") {
			t.Errorf("expected 'Sources' in response, got: %s", res.HTML(t, 200))
		}
	})
}

func TestSourceController_New(t *testing.T) {
	t.Run("renders form", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/sources/new")

		html := res.HTML(t, 200)
		if !strings.Contains(html, "New Source") {
			t.Errorf("expected 'New Source' in response")
		}
	})

	t.Run("renders correct layout when onboarding", func(t *testing.T) {
		c := webtest.New(t)
		c.App.SettingsSet(c.Ctx, core.KW{core.Opt("onboarding", true)})

		res := c.Get("/sources/new")

		html := res.HTML(t, 200)
		if strings.Contains(html, "Main navigation") {
			t.Errorf("should not show Main navigation when onboarding")
		}
	})

	t.Run("preloads some attributes when using a template", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{
			"custom_name":          "My first source",
			"download_cutoff_date": "2021-01-01",
		})

		res := c.Get("/sources/new", map[string]string{"template_id": fmt.Sprint(source.ID)})

		html := res.HTML(t, 200)
		if !strings.Contains(html, "New Source") {
			t.Errorf("expected 'New Source' in response")
		}
		if !strings.Contains(html, "2021-01-01") {
			t.Errorf("expected '2021-01-01' in response")
		}
		if strings.Contains(html, source.CustomName) {
			t.Errorf("should not show template custom name")
		}
	})
}

func TestSourceController_Create(t *testing.T) {
	t.Run("redirects to show when data is valid", func(t *testing.T) {
		c := webtest.New(t)
		profile := coretest.MediaProfileFixture(t, c.TestApp, core.Attrs{})
		c.YtDlpMock.Run.ExpectN(1, func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "{\"channel\":\"test\",\"channel_id\":\"ch123\",\"playlist_id\":\"pl123\",\"playlist_title\":\"test\"}", nil
		})

		res := c.Post("/sources", "source", core.Attrs{
			"media_profile_id": profile.ID,
			"collection_type":  "channel",
			"original_url":     "https://www.youtube.com/source/abc123",
		})

		if res.Status != 302 {
			t.Errorf("expected 302 redirect, got %d", res.Status)
		}
		if !strings.HasPrefix(res.RedirectedTo(t), "/sources/") {
			t.Errorf("expected redirect to /sources/*, got %s", res.RedirectedTo(t))
		}
	})

	t.Run("renders errors when data is invalid", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Post("/sources", "source", core.Attrs{
			"original_url":     nil,
			"media_profile_id": nil,
		})

		html := res.HTML(t, 200)
		if !strings.Contains(html, "New Source") {
			t.Errorf("expected 'New Source' in error response")
		}
	})

	t.Run("redirects to onboarding when onboarding", func(t *testing.T) {
		c := webtest.New(t)
		c.App.SettingsSet(c.Ctx, core.KW{core.Opt("onboarding", true)})
		profile := coretest.MediaProfileFixture(t, c.TestApp, core.Attrs{})
		c.YtDlpMock.Run.ExpectN(1, func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "{\"channel\":\"test\",\"channel_id\":\"ch123\",\"playlist_id\":\"pl123\",\"playlist_title\":\"test\"}", nil
		})

		res := c.Post("/sources", "source", core.Attrs{
			"media_profile_id": profile.ID,
			"collection_type":  "channel",
			"original_url":     "https://www.youtube.com/source/abc123",
		})

		if res.RedirectedTo(t) != "/?onboarding=1" {
			t.Errorf("expected redirect to /?onboarding=1, got %s", res.RedirectedTo(t))
		}
	})

	t.Run("renders correct layout on error when onboarding", func(t *testing.T) {
		c := webtest.New(t)
		c.App.SettingsSet(c.Ctx, core.KW{core.Opt("onboarding", true)})

		res := c.Post("/sources", "source", core.Attrs{
			"original_url":     nil,
			"media_profile_id": nil,
		})

		html := res.HTML(t, 200)
		if strings.Contains(html, "Main navigation") {
			t.Errorf("should not show Main navigation when onboarding")
		}
	})
}

func TestSourceController_Edit(t *testing.T) {
	t.Run("renders form for editing chosen source", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Get(fmt.Sprintf("/sources/%d/edit", source.ID))

		html := res.HTML(t, 200)
		if !strings.Contains(html, fmt.Sprintf("Editing \"%s\"", source.CustomName)) {
			t.Errorf("expected edit heading for source")
		}
	})
}

func TestSourceController_Update(t *testing.T) {
	t.Run("redirects when data is valid", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
		c.YtDlpMock.Run.ExpectN(1, func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) {
			return "{\"channel\":\"test\",\"channel_id\":\"ch123\",\"playlist_id\":\"pl123\",\"playlist_title\":\"test\"}", nil
		})

		res := c.Patch(
			fmt.Sprintf("/sources/%d", source.ID),
			"source",
			core.Attrs{"original_url": "https://www.youtube.com/source/321xyz"},
		)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to /sources/%d", source.ID)
		}

		// Verify the update happened
		res = c.Get(fmt.Sprintf("/sources/%d", source.ID))
		html := res.HTML(t, 200)
		if !strings.Contains(html, "https://www.youtube.com/source/321xyz") {
			t.Errorf("expected updated URL in response")
		}
	})

	t.Run("renders errors when data is invalid", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Patch(
			fmt.Sprintf("/sources/%d", source.ID),
			"source",
			core.Attrs{"original_url": nil, "media_profile_id": nil},
		)

		html := res.HTML(t, 200)
		if !strings.Contains(html, fmt.Sprintf("Editing \"%s\"", source.CustomName)) {
			t.Errorf("expected edit form with errors")
		}
	})
}

func TestSourceController_Delete(t *testing.T) {
	t.Run("delete source in all cases - redirects to the sources page", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Delete(fmt.Sprintf("/sources/%d", source.ID))

		if res.RedirectedTo(t) != "/sources" {
			t.Errorf("expected redirect to /sources, got %s", res.RedirectedTo(t))
		}
	})

	t.Run("delete source in all cases - sets marked_for_deletion_at", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		c.Delete(fmt.Sprintf("/sources/%d", source.ID))

		reloaded, _ := c.App.SourcesGetSource(c.Ctx, source.ID)
		if reloaded.MarkedForDeletionAt == nil {
			t.Errorf("expected marked_for_deletion_at to be set")
		}
	})

	t.Run("delete source when just deleting the records - enqueues a job without the delete_files arg", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		c.Delete(fmt.Sprintf("/sources/%d", source.ID))

		jobs := c.Oban.Enqueued(t, obanlite.Match{Args: map[string]any{"delete_files": false}})
		if len(jobs) == 0 {
			t.Errorf("expected job to be enqueued with delete_files=false")
		}
	})

	t.Run("delete source when deleting the records and files - enqueues a job with the delete_files arg", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		c.Delete(fmt.Sprintf("/sources/%d?delete_files=true", source.ID))

		jobs := c.Oban.Enqueued(t, obanlite.Match{Args: map[string]any{"delete_files": true}})
		if len(jobs) == 0 {
			t.Errorf("expected job to be enqueued with delete_files=true")
		}
	})
}

func TestSourceController_ForceDownloadPending(t *testing.T) {
	t.Run("enqueues pending download tasks", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
		coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID, "media_filepath": nil})

		initialJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))
		c.Post(fmt.Sprintf("/sources/%d/force_download_pending", source.ID), "", nil)
		finalJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))

		if finalJobs <= initialJobs {
			t.Errorf("expected job to be enqueued")
		}
	})

	t.Run("redirects to the source page", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/force_download_pending", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}

func TestSourceController_ForceRedownload(t *testing.T) {
	t.Run("enqueues re-download tasks", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
		coretest.MediaItemFixture(t, c.TestApp, core.Attrs{
			"source_id":           source.ID,
			"media_downloaded_at": coretest.Now(),
		})

		initialJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))
		c.Post(fmt.Sprintf("/sources/%d/force_redownload", source.ID), "", nil)
		finalJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))

		if finalJobs <= initialJobs {
			t.Errorf("expected job to be enqueued")
		}
	})

	t.Run("redirects to the source page", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/force_redownload", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}

func TestSourceController_ForceIndex(t *testing.T) {
	t.Run("forces an index", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		initialJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))
		c.Post(fmt.Sprintf("/sources/%d/force_index", source.ID), "", nil)
		finalJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))

		if finalJobs <= initialJobs {
			t.Errorf("expected job to be enqueued")
		}
	})

	t.Run("forces an index even if one wouldn't normally run", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{
			"index_frequency_minutes": int64(0),
			"last_indexed_at":         coretest.Now(),
		})

		c.Post(fmt.Sprintf("/sources/%d/force_index", source.ID), "", nil)

		jobs := c.Oban.Enqueued(t, obanlite.Match{Args: map[string]any{"id": source.ID, "force": true}})
		if len(jobs) == 0 {
			t.Errorf("expected job to be enqueued with force=true")
		}
	})

	t.Run("redirects to the source page", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/force_index", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}

func TestSourceController_ForceMetadataRefresh(t *testing.T) {
	t.Run("forces a metadata refresh", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		initialJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))
		c.Post(fmt.Sprintf("/sources/%d/force_metadata_refresh", source.ID), "", nil)
		finalJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))

		if finalJobs <= initialJobs {
			t.Errorf("expected job to be enqueued")
		}
	})

	t.Run("redirects to the source page", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/force_metadata_refresh", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}

func TestSourceController_SyncFilesOnDisk(t *testing.T) {
	t.Run("forces a file sync", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		initialJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))
		c.Post(fmt.Sprintf("/sources/%d/sync_files_on_disk", source.ID), "", nil)
		finalJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))

		if finalJobs <= initialJobs {
			t.Errorf("expected job to be enqueued")
		}
	})

	t.Run("redirects to the source page", func(t *testing.T) {
		c := webtest.New(t)
		source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/sync_files_on_disk", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}
