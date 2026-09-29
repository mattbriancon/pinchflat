package web_test

import (
	"strconv"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

// Port of test/pinchflat_web/controllers/pages/job_table_live_test.exs. The
// job table is now rendered inline on the home page ("/") instead of the
// old /_live/jobs fragment (STRATEGY.md decision 4: no htmx).

// createMediaItemJob is create_media_item_job/1.
func createMediaItemJob(t testing.TB, c *webtest.Client, jobState string) (*store.Source, *store.MediaItem, *store.Task) {
	t.Helper()
	source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})
	mediaItem := apptest.MediaItemFixture(t, c.TestApp, store.Attrs{"source_id": source.ID})
	task, err := c.App.MediaDownloadWorkerKickoffWithTask(c.Ctx, mediaItem, store.Attrs{}, nil)
	if err != nil {
		t.Fatalf("MediaDownloadWorkerKickoffWithTask: %v", err)
	}

	_, err = store.Exec(c.Ctx, c.App.Q(c.Ctx), store.SQ.Update("oban_jobs").Set("state", jobState).Where(sq.Eq{"id": task.JobID}))
	if err != nil {
		t.Fatalf("failed to update job state: %v", err)
	}

	return source, mediaItem, task
}

// createSourceJob is create_source_job/1.
func createSourceJob(t testing.TB, c *webtest.Client, jobState string) (*store.Source, *store.Task) {
	t.Helper()
	source := apptest.SourceFixture(t, c.TestApp, store.Attrs{})
	task, err := c.App.FastIndexingWorkerKickoffWithTask(c.Ctx, source, 0)
	if err != nil {
		t.Fatalf("FastIndexingWorkerKickoffWithTask: %v", err)
	}

	_, err = store.Exec(c.Ctx, c.App.Q(c.Ctx), store.SQ.Update("oban_jobs").Set("state", jobState).Where(sq.Eq{"id": task.JobID}))
	if err != nil {
		t.Fatalf("failed to update job state: %v", err)
	}

	return source, task
}

// homeClient is a client past onboarding, ready to render the home page.
func homeClient(t testing.TB) *webtest.Client {
	t.Helper()
	c := webtest.New(t)
	if _, err := c.App.SetSetting(c.Ctx, store.KW{store.Opt("onboarding", false)}); err != nil {
		t.Fatalf("SettingsSet: %v", err)
	}
	return c
}

func TestJobTableLive_InitialRendering(t *testing.T) {
	t.Run("shows message when no records", func(t *testing.T) {
		c := homeClient(t)
		res := c.Get("/")
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Nothing Here!") {
			t.Errorf("expected html to contain %q, got: %s", "Nothing Here!", html)
		}
		if strings.Contains(html, "Subject") {
			t.Errorf("expected html not to contain %q, got: %s", "Subject", html)
		}
	})

	t.Run("shows records when present", func(t *testing.T) {
		c := homeClient(t)
		createMediaItemJob(t, c, "executing")

		res := c.Get("/")
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Subject") {
			t.Errorf("expected html to contain %q, got: %s", "Subject", html)
		}
	})

	t.Run("doesn't show records when not in executing state", func(t *testing.T) {
		c := homeClient(t)
		createMediaItemJob(t, c, "scheduled")
		createMediaItemJob(t, c, "completed")

		res := c.Get("/")
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Nothing Here!") {
			t.Errorf("expected html to contain %q, got: %s", "Nothing Here!", html)
		}
		if strings.Contains(html, "Subject") {
			t.Errorf("expected html not to contain %q, got: %s", "Subject", html)
		}
	})
}

func TestJobTableLive_JobRendering(t *testing.T) {
	t.Run("shows worker name", func(t *testing.T) {
		c := homeClient(t)
		createMediaItemJob(t, c, "executing")

		res := c.Get("/")
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Downloading Media") {
			t.Errorf("expected html to contain %q, got: %s", "Downloading Media", html)
		}
	})

	t.Run("shows the media item title", func(t *testing.T) {
		c := homeClient(t)
		_, mediaItem, _ := createMediaItemJob(t, c, "executing")

		res := c.Get("/")
		html := res.HTML(t, 200)

		if !strings.Contains(html, *mediaItem.Title) {
			t.Errorf("expected html to contain %q, got: %s", *mediaItem.Title, html)
		}
	})

	t.Run("shows a media item link", func(t *testing.T) {
		c := homeClient(t)
		_, mediaItem, _ := createMediaItemJob(t, c, "executing")

		res := c.Get("/")
		html := res.HTML(t, 200)

		want := "/sources/" + strconv.FormatInt(mediaItem.SourceID, 10) + "/media/" + strconv.FormatInt(mediaItem.ID, 10)
		if !strings.Contains(html, want) {
			t.Errorf("expected html to contain %q, got: %s", want, html)
		}
	})

	t.Run("shows the source custom name", func(t *testing.T) {
		c := homeClient(t)
		source, _ := createSourceJob(t, c, "executing")

		res := c.Get("/")
		html := res.HTML(t, 200)

		if !strings.Contains(html, source.CustomName) {
			t.Errorf("expected html to contain %q, got: %s", source.CustomName, html)
		}
	})

	t.Run("shows a source link", func(t *testing.T) {
		c := homeClient(t)
		source, _ := createSourceJob(t, c, "executing")

		res := c.Get("/")
		html := res.HTML(t, 200)

		want := "/sources/" + strconv.FormatInt(source.ID, 10)
		if !strings.Contains(html, want) {
			t.Errorf("expected html to contain %q, got: %s", want, html)
		}
	})

	t.Run("listens for job:state change events", func(t *testing.T) {
		// PubSub broadcast -> re-render has no Go equivalent, and there is no
		// server push or refresh button in the no-htmx port (STRATEGY.md): a
		// plain browser reload of "/" is the replacement, which the other
		// tests here exercise.
		t.Skip("DROPPED: no server push; a reload of the page replaces it")
	})
}
