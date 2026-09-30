package web

import (
	"context"
	"errors"
	"net/http"

	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// TaskControllerRetry runs a task's scheduled or retryable job now instead of
// waiting out its schedule or retry backoff, then goes back to the page the
// button was on.
func (s *Server) TaskControllerRetry(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	task, ok := loadOrFail(s, w, r, "id", s.App.GetTaskBang)
	if !ok {
		return
	}
	for _, preload := range []func(context.Context, *store.Task) (*store.Task, error){
		s.App.PreloadTaskJob, s.App.PreloadTaskSource, s.App.PreloadTaskMediaItem,
	} {
		if _, err := preload(ctx, task); err != nil {
			s.Fail(w, r, err)
			return
		}
	}

	if !taskRetryable(task) {
		s.PutFlash(w, r, "error", "Task isn't waiting to run, so it can't be retried now.")
		s.redirectBack(w, r, taskToLink(ctx, task))
		return
	}

	if err := s.App.Oban.RetryJob(ctx, task.JobID); err != nil {
		if !errors.Is(err, obanlite.ErrNotRetryable) {
			s.Fail(w, r, err)
			return
		}
		s.PutFlash(w, r, "error", "Task isn't waiting to run, so it can't be retried now.")
	} else {
		s.PutFlash(w, r, "info", "Task will run now.")
	}
	s.redirectBack(w, r, taskToLink(ctx, task))
}
