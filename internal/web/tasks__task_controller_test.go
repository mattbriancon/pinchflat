package web_test

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

const farFuture = "2999-01-01T00:00:00.000000Z"

// waitingTask is a task on a new source whose job is in state, scheduled far in
// the future.
func waitingTask(t *testing.T, c *webtest.Client, state string, attempt, maxAttempts int) *store.Task {
	t.Helper()
	task := apptest.TaskFixture(t, c.TestApp, apptest.TaskParams{})
	_, err := c.App.Oban.DB().Exec(`UPDATE oban_jobs SET state = ?, attempt = ?, max_attempts = ?, scheduled_at = ? WHERE id = ?`,
		state, attempt, maxAttempts, farFuture, task.JobID)
	must(t, err)
	return task
}

func TestTaskController_Retry(t *testing.T) {
	for _, state := range []string{"scheduled", "retryable"} {
		t.Run("runs a "+state+" job now and redirects back", func(t *testing.T) {
			c := webtest.New(t)
			task := waitingTask(t, c, state, 3, 20)

			req := httptest.NewRequest("POST", fmt.Sprintf("/tasks/%d/retry", task.ID), nil)
			req.Header.Set("Referer", fmt.Sprintf("http://example.com/sources/%d", *task.SourceID))
			res := c.Do(req)

			if res.Status != 303 {
				t.Fatalf("expected a 303, got %d", res.Status)
			}
			if got := res.Header.Get("Location"); got != fmt.Sprintf("/sources/%d", *task.SourceID) {
				t.Errorf("redirected to %q", got)
			}
			if got := res.Flash("info"); got != "Task will run now." {
				t.Errorf("flash %q", got)
			}
			job, err := c.App.Oban.GetJob(c.Ctx, task.JobID)
			must(t, err)
			if job.State != "available" || job.ScheduledAt.Time.After(apptest.Now()) {
				t.Errorf("job state %s, scheduled_at %v", job.State, job.ScheduledAt.Time)
			}
		})
	}

	t.Run("refuses a job that isn't waiting", func(t *testing.T) {
		c := webtest.New(t)
		task := waitingTask(t, c, "executing", 1, 20)

		res := c.Post(fmt.Sprintf("/tasks/%d/retry", task.ID), "", nil)

		if res.Status != 303 {
			t.Fatalf("expected a 303, got %d", res.Status)
		}
		if res.Flash("error") == "" {
			t.Errorf("expected an error flash")
		}
		job, _ := c.App.Oban.GetJob(c.Ctx, task.JobID)
		if job.State != "executing" {
			t.Errorf("job state %s", job.State)
		}
	})

	t.Run("404s for a missing task", func(t *testing.T) {
		c := webtest.New(t)
		if res := c.Post("/tasks/99999/retry", "", nil); res.Status != 404 {
			t.Errorf("expected 404, got %d", res.Status)
		}
	})
}

func TestTaskTables(t *testing.T) {
	t.Run("source page shows next run, attempt and a retry button", func(t *testing.T) {
		c := webtest.New(t)
		task := waitingTask(t, c, "retryable", 12, 20)

		body := c.Get(fmt.Sprintf("/sources/%d", *task.SourceID)).HTML(t, 200)

		for _, want := range []string{"retryable (attempt 12 of 20)", "2999-01-01 00:00:00", fmt.Sprintf("/tasks/%d/retry", task.ID), "Retry now"} {
			if !strings.Contains(body, want) {
				t.Errorf("expected page to contain %q", want)
			}
		}
	})

	t.Run("source page has no retry button for an executing job", func(t *testing.T) {
		c := webtest.New(t)
		task := waitingTask(t, c, "executing", 1, 20)

		body := c.Get(fmt.Sprintf("/sources/%d", *task.SourceID)).HTML(t, 200)

		if strings.Contains(body, "Retry now") {
			t.Errorf("unexpected retry button")
		}
	})

	t.Run("media item page shows next run and a retry button", func(t *testing.T) {
		c := webtest.New(t)
		m := createMediaItem(t, c)
		job := apptest.JobFixture(t, c.TestApp)
		task := apptest.TaskFixture(t, c.TestApp, apptest.TaskParams{JobID: store.Ptr(job.ID), SourceID: store.Ptr(m.SourceID), MediaItemID: store.Ptr(m.ID)})
		_, err := c.App.Oban.DB().Exec(`UPDATE oban_jobs SET state = 'retryable', attempt = 2, max_attempts = 5, scheduled_at = ? WHERE id = ?`, farFuture, job.ID)
		must(t, err)

		body := c.Get(mediaPath(m, "")).HTML(t, 200)

		for _, want := range []string{"retryable (attempt 2 of 5)", "2999-01-01 00:00:00", fmt.Sprintf("/tasks/%d/retry", task.ID)} {
			if !strings.Contains(body, want) {
				t.Errorf("expected page to contain %q", want)
			}
		}
	})
}
