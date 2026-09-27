package web_test

import (
	"strconv"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

// Port of test/pinchflat_web/controllers/pages/job_table_live_test.exs.

// createMediaItemJob is create_media_item_job/1.
func createMediaItemJob(t testing.TB, c *webtest.Client, jobState string) (*core.Source, *core.MediaItem, *core.Task) {
	t.Helper()
	source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
	mediaItem := coretest.MediaItemFixture(t, c.TestApp, core.Attrs{"source_id": source.ID})
	task, err := c.App.MediaDownloadWorkerKickoffWithTask(c.Ctx, mediaItem, core.Attrs{}, core.KW{})
	if err != nil {
		t.Fatalf("MediaDownloadWorkerKickoffWithTask: %v", err)
	}

	_, err = core.Exec(c.Ctx, c.App.Q(c.Ctx), core.SQ.Update("oban_jobs").Set("state", jobState).Where(sq.Eq{"id": task.JobID}))
	if err != nil {
		t.Fatalf("failed to update job state: %v", err)
	}

	return source, mediaItem, task
}

// createSourceJob is create_source_job/1.
func createSourceJob(t testing.TB, c *webtest.Client, jobState string) (*core.Source, *core.Task) {
	t.Helper()
	source := coretest.SourceFixture(t, c.TestApp, core.Attrs{})
	task, err := c.App.FastIndexingWorkerKickoffWithTask(c.Ctx, source, core.KW{})
	if err != nil {
		t.Fatalf("FastIndexingWorkerKickoffWithTask: %v", err)
	}

	_, err = core.Exec(c.Ctx, c.App.Q(c.Ctx), core.SQ.Update("oban_jobs").Set("state", jobState).Where(sq.Eq{"id": task.JobID}))
	if err != nil {
		t.Fatalf("failed to update job state: %v", err)
	}

	return source, task
}

func TestJobTableLive_InitialRendering(t *testing.T) {
	t.Run("shows message when no records", func(t *testing.T) {
		c := webtest.New(t)
		res := c.Get("/_live/jobs")
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Nothing Here!") {
			t.Errorf("expected html to contain %q, got: %s", "Nothing Here!", html)
		}
		if strings.Contains(html, "Subject") {
			t.Errorf("expected html not to contain %q, got: %s", "Subject", html)
		}
	})

	t.Run("shows records when present", func(t *testing.T) {
		c := webtest.New(t)
		createMediaItemJob(t, c, "executing")

		res := c.Get("/_live/jobs")
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Subject") {
			t.Errorf("expected html to contain %q, got: %s", "Subject", html)
		}
	})

	t.Run("doesn't show records when not in executing state", func(t *testing.T) {
		c := webtest.New(t)
		createMediaItemJob(t, c, "scheduled")
		createMediaItemJob(t, c, "completed")

		res := c.Get("/_live/jobs")
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
		c := webtest.New(t)
		createMediaItemJob(t, c, "executing")

		res := c.Get("/_live/jobs")
		html := res.HTML(t, 200)

		if !strings.Contains(html, "Downloading Media") {
			t.Errorf("expected html to contain %q, got: %s", "Downloading Media", html)
		}
	})

	t.Run("shows the media item title", func(t *testing.T) {
		c := webtest.New(t)
		_, mediaItem, _ := createMediaItemJob(t, c, "executing")

		res := c.Get("/_live/jobs")
		html := res.HTML(t, 200)

		if !strings.Contains(html, *mediaItem.Title) {
			t.Errorf("expected html to contain %q, got: %s", *mediaItem.Title, html)
		}
	})

	t.Run("shows a media item link", func(t *testing.T) {
		c := webtest.New(t)
		_, mediaItem, _ := createMediaItemJob(t, c, "executing")

		res := c.Get("/_live/jobs")
		html := res.HTML(t, 200)

		want := "/sources/" + strconv.FormatInt(mediaItem.SourceID, 10) + "/media/" + strconv.FormatInt(mediaItem.ID, 10)
		if !strings.Contains(html, want) {
			t.Errorf("expected html to contain %q, got: %s", want, html)
		}
	})

	t.Run("shows the source custom name", func(t *testing.T) {
		c := webtest.New(t)
		source, _ := createSourceJob(t, c, "executing")

		res := c.Get("/_live/jobs")
		html := res.HTML(t, 200)

		if !strings.Contains(html, source.CustomName) {
			t.Errorf("expected html to contain %q, got: %s", source.CustomName, html)
		}
	})

	t.Run("shows a source link", func(t *testing.T) {
		c := webtest.New(t)
		source, _ := createSourceJob(t, c, "executing")

		res := c.Get("/_live/jobs")
		html := res.HTML(t, 200)

		want := "/sources/" + strconv.FormatInt(source.ID, 10)
		if !strings.Contains(html, want) {
			t.Errorf("expected html to contain %q, got: %s", want, html)
		}
	})

	// "listens for job:state change events" (PubSub broadcast -> re-render) has
	// no Go equivalent: there is no server push in the htmx port (see
	// STRATEGY.md / CONVENTIONS "LiveViews -> htmx"); the fragment endpoint is
	// instead re-fetched by a manual "Reload" button, which the other tests in
	// this file already exercise by re-GETting /_live/jobs.
}
