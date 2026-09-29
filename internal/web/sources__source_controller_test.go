package web_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
	"github.com/mattbriancon/pinchflat/internal/ytdlp"
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
		c.App.SetSetting(c.Ctx, store.KW{store.Opt("onboarding", true)})

		res := c.Get("/sources/new")

		html := res.HTML(t, 200)
		if strings.Contains(html, "Main navigation") {
			t.Errorf("should not show Main navigation when onboarding")
		}
	})

	t.Run("preloads some attributes when using a template", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{
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
		c.App.SetSetting(c.Ctx, store.KW{store.Opt("onboarding", false)})
		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})
		c.YtDlpMock.Run.ExpectN(1, func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return "{\"channel\":\"test\",\"channel_id\":\"ch123\",\"playlist_id\":\"pl123\",\"playlist_title\":\"test\"}", nil
		})

		res := c.Post("/sources", "source", store.Attrs{
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

		res := c.Post("/sources", "source", store.Attrs{
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
		c.App.SetSetting(c.Ctx, store.KW{store.Opt("onboarding", true)})
		profile := apptest.MediaProfileFixture(t, c.TestApp, store.MediaProfileParams{})
		c.YtDlpMock.Run.ExpectN(1, func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return "{\"channel\":\"test\",\"channel_id\":\"ch123\",\"playlist_id\":\"pl123\",\"playlist_title\":\"test\"}", nil
		})

		res := c.Post("/sources", "source", store.Attrs{
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
		c.App.SetSetting(c.Ctx, store.KW{store.Opt("onboarding", true)})

		res := c.Post("/sources", "source", store.Attrs{
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
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

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
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})
		c.YtDlpMock.Run.ExpectN(1, func(url, action string, opts ytdlp.Args, ot string, addl ytdlp.CallOptions) (string, error) {
			return "{\"channel\":\"test\",\"channel_id\":\"ch123\",\"playlist_id\":\"pl123\",\"playlist_title\":\"test\"}", nil
		})

		res := c.Patch(
			fmt.Sprintf("/sources/%d", source.ID),
			"source",
			store.Attrs{"original_url": "https://www.youtube.com/source/321xyz"},
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
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		res := c.Patch(
			fmt.Sprintf("/sources/%d", source.ID),
			"source",
			store.Attrs{"original_url": nil, "media_profile_id": nil},
		)

		html := res.HTML(t, 200)
		if !strings.Contains(html, fmt.Sprintf("Editing \"%s\"", source.CustomName)) {
			t.Errorf("expected edit form with errors")
		}
	})
}

func TestSourceController_Delete_InAllCases(t *testing.T) {
	t.Run("redirects to the sources page", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		res := c.Delete(fmt.Sprintf("/sources/%d", source.ID))

		if res.RedirectedTo(t) != "/sources" {
			t.Errorf("expected redirect to /sources, got %s", res.RedirectedTo(t))
		}
	})

	t.Run("sets marked_for_deletion_at", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		c.Delete(fmt.Sprintf("/sources/%d", source.ID))

		reloaded, _ := c.App.GetSource(c.Ctx, source.ID)
		if reloaded.MarkedForDeletionAt == nil {
			t.Errorf("expected marked_for_deletion_at to be set")
		}
	})
}

func TestSourceController_Delete_JustDeletingRecords(t *testing.T) {
	t.Run("enqueues a job without the delete_files arg", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		c.Delete(fmt.Sprintf("/sources/%d", source.ID))

		jobs := c.Oban.Enqueued(t, obanlite.Match{Args: map[string]any{"delete_files": false}})
		if len(jobs) == 0 {
			t.Errorf("expected job to be enqueued with delete_files=false")
		}
	})
}

func TestSourceController_Delete_DeletingRecordsAndFiles(t *testing.T) {
	t.Run("enqueues a job without the delete_files arg", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

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
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})
		apptest.MediaItemFixture(t, c.TestApp, store.MediaItemParams{SourceID: store.Ptr(source.ID), Clear: store.ClearMediaFilepath})

		initialJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))
		c.Post(fmt.Sprintf("/sources/%d/force_download_pending", source.ID), "", nil)
		finalJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))

		if finalJobs <= initialJobs {
			t.Errorf("expected job to be enqueued")
		}
	})

	t.Run("redirects to the source page", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/force_download_pending", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}

func TestSourceController_ForceRedownload(t *testing.T) {
	t.Run("enqueues re-download tasks", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})
		apptest.MediaItemFixture(t, c.TestApp, store.MediaItemParams{
			SourceID:          store.Ptr(source.ID),
			MediaDownloadedAt: store.Ptr(apptest.Now()),
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
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/force_redownload", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}

func TestSourceController_ForceIndex(t *testing.T) {
	t.Run("forces an index", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		initialJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))
		c.Post(fmt.Sprintf("/sources/%d/force_index", source.ID), "", nil)
		finalJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))

		if finalJobs <= initialJobs {
			t.Errorf("expected job to be enqueued")
		}
	})

	t.Run("forces an index even if one wouldn't normally run", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{
			"index_frequency_minutes": int64(0),
			"last_indexed_at":         apptest.Now(),
		})

		c.Post(fmt.Sprintf("/sources/%d/force_index", source.ID), "", nil)

		jobs := c.Oban.Enqueued(t, obanlite.Match{Args: map[string]any{"id": source.ID, "force": true}})
		if len(jobs) == 0 {
			t.Errorf("expected job to be enqueued with force=true")
		}
	})

	t.Run("deletes pending indexing tasks", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})
		task, err := c.App.SlowIndexingHelpersKickoffIndexingTask(c.Ctx, source, store.Attrs{})
		if err != nil {
			t.Fatalf("kickoff failed: %v", err)
		}
		task, _ = c.App.PreloadTaskJob(c.Ctx, task)

		if task.Job.State != "available" {
			t.Fatalf("expected job state 'available', got %s", task.Job.State)
		}

		c.Post(fmt.Sprintf("/sources/%d/force_index", source.ID), "", nil)

		reloadedJob, err := c.Oban.GetJob(c.Ctx, task.Job.ID)
		if err != nil {
			t.Fatalf("reload job failed: %v", err)
		}
		if reloadedJob.State != "cancelled" {
			t.Errorf("expected job state 'cancelled', got %s", reloadedJob.State)
		}
	})

	t.Run("redirects to the source page", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/force_index", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}

func TestSourceController_ForceMetadataRefresh(t *testing.T) {
	t.Run("forces a metadata refresh", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		initialJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))
		c.Post(fmt.Sprintf("/sources/%d/force_metadata_refresh", source.ID), "", nil)
		finalJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))

		if finalJobs <= initialJobs {
			t.Errorf("expected job to be enqueued")
		}
	})

	t.Run("redirects to the source page", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/force_metadata_refresh", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}

func TestSourceController_SyncFilesOnDisk(t *testing.T) {
	t.Run("forces a file sync", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		initialJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))
		c.Post(fmt.Sprintf("/sources/%d/sync_files_on_disk", source.ID), "", nil)
		finalJobs := len(c.Oban.Enqueued(t, obanlite.Match{}))

		if finalJobs <= initialJobs {
			t.Errorf("expected job to be enqueued")
		}
	})

	t.Run("redirects to the source page", func(t *testing.T) {
		c := webtest.New(t)
		source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})

		res := c.Post(fmt.Sprintf("/sources/%d/sync_files_on_disk", source.ID), "", nil)

		if res.RedirectedTo(t) != fmt.Sprintf("/sources/%d", source.ID) {
			t.Errorf("expected redirect to source page")
		}
	})
}
