package core

// Port of lib/pinchflat/repo.ex (the helpers on Pinchflat.Repo).
// Hand-written W0 infrastructure.

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// InsertUniqueJob inserts a job, reporting whether it was a duplicate of an
// existing unique job (Repo.insert_unique_job/1: {:ok, job} | {:duplicate,
// job} | {:error, _}).
func (a *App) InsertUniqueJob(ctx context.Context, spec obanlite.JobSpec) (*obanlite.Job, bool, error) {
	job, err := a.Oban.Insert(ctx, a.Q(ctx), spec)
	if err != nil {
		return nil, false, err
	}
	return job, job.Conflict, nil
}

// MaybeLimit applies a limit when limit is non-nil (Repo.maybe_limit/2).
func MaybeLimit(q sq.SelectBuilder, limit *int) sq.SelectBuilder {
	if limit != nil {
		return q.Limit(uint64(*limit))
	}
	return q
}
