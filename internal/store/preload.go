package store

// Association loading: the Repo.preload/2 calls Pinchflat makes. Each helper
// always (re)loads, like `force: true`, and returns the same pointer for
// chaining. Hand-written W0 infrastructure.

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// PreloadSourceMediaProfile loads source.media_profile.
func (st *Store) PreloadSourceMediaProfile(ctx context.Context, s *Source) (*Source, error) {
	p, err := Get[MediaProfile](ctx, st.Q(ctx), s.MediaProfileID)
	if err != nil {
		return s, err
	}
	s.MediaProfile = p
	return s, nil
}

// PreloadSourceMetadata loads source.metadata (nil if none).
func (st *Store) PreloadSourceMetadata(ctx context.Context, s *Source) (*Source, error) {
	m, err := One[SourceMetadata](ctx, st.Q(ctx), From[SourceMetadata]().Where(sq.Eq{"source_id": s.ID}))
	if err != nil {
		return s, err
	}
	s.Metadata = m
	return s, nil
}

// PreloadSourceMediaItems loads source.media_items ordered by id.
func (st *Store) PreloadSourceMediaItems(ctx context.Context, s *Source) (*Source, error) {
	items, err := All[MediaItem](ctx, st.Q(ctx), From[MediaItem]().Where(sq.Eq{"source_id": s.ID}).OrderBy("id"))
	if err != nil {
		return s, err
	}
	s.MediaItems = items
	return s, nil
}

// PreloadSourceTasks loads source.tasks with each task's job
// (preload(tasks: [:job])).
func (st *Store) PreloadSourceTasks(ctx context.Context, s *Source) (*Source, error) {
	tasks, err := All[Task](ctx, st.Q(ctx), From[Task]().Where(sq.Eq{"source_id": s.ID}).OrderBy("id"))
	if err != nil {
		return s, err
	}
	if err := st.PreloadTasksJobs(ctx, tasks); err != nil {
		return s, err
	}
	s.Tasks = tasks
	return s, nil
}

// PreloadMediaItemSource loads media_item.source.
func (st *Store) PreloadMediaItemSource(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	s, err := Get[Source](ctx, st.Q(ctx), m.SourceID)
	if err != nil {
		return m, err
	}
	m.Source = s
	return m, nil
}

// PreloadMediaItemSourceAndProfile loads media_item.source.media_profile
// (preload(source: :media_profile)).
func (st *Store) PreloadMediaItemSourceAndProfile(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	if _, err := st.PreloadMediaItemSource(ctx, m); err != nil {
		return m, err
	}
	_, err := st.PreloadSourceMediaProfile(ctx, m.Source)
	return m, err
}

// PreloadMediaItemMetadata loads media_item.metadata (nil if none).
func (st *Store) PreloadMediaItemMetadata(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	md, err := One[MediaMetadata](ctx, st.Q(ctx), From[MediaMetadata]().Where(sq.Eq{"media_item_id": m.ID}))
	if err != nil {
		return m, err
	}
	m.Metadata = md
	return m, nil
}

// PreloadMediaItemTasks loads media_item.tasks with their jobs.
func (st *Store) PreloadMediaItemTasks(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	tasks, err := All[Task](ctx, st.Q(ctx), From[Task]().Where(sq.Eq{"media_item_id": m.ID}).OrderBy("id"))
	if err != nil {
		return m, err
	}
	if err := st.PreloadTasksJobs(ctx, tasks); err != nil {
		return m, err
	}
	m.Tasks = tasks
	return m, nil
}

// PreloadMediaItemFull loads [:metadata, source: :media_profile].
func (st *Store) PreloadMediaItemFull(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	if _, err := st.PreloadMediaItemMetadata(ctx, m); err != nil {
		return m, err
	}
	return st.PreloadMediaItemSourceAndProfile(ctx, m)
}

// PreloadMediaProfileSources loads media_profile.sources.
func (st *Store) PreloadMediaProfileSources(ctx context.Context, p *MediaProfile) (*MediaProfile, error) {
	sources, err := All[Source](ctx, st.Q(ctx), From[Source]().Where(sq.Eq{"media_profile_id": p.ID}).OrderBy("id"))
	if err != nil {
		return p, err
	}
	p.Sources = sources
	return p, nil
}

// PreloadTaskJob loads task.job.
func (st *Store) PreloadTaskJob(ctx context.Context, t *Task) (*Task, error) {
	var job obanlite.Job
	err := st.Q(ctx).GetContext(ctx, &job, `SELECT id, state, queue, worker, args, meta, tags, errors, attempt, max_attempts, priority,
		inserted_at, scheduled_at, attempted_at, attempted_by, cancelled_at, completed_at, discarded_at FROM oban_jobs WHERE id = ?`, t.JobID)
	if err != nil {
		return t, err
	}
	t.Job = &job
	return t, nil
}

// PreloadTasksJobs loads jobs for multiple tasks in one query.
func (st *Store) PreloadTasksJobs(ctx context.Context, tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}

	// Collect job IDs from tasks
	jobIDs := make([]interface{}, len(tasks))
	for i, t := range tasks {
		jobIDs[i] = t.JobID
	}

	// Load all jobs in one query
	jobs, err := All[obanlite.Job](ctx, st.Q(ctx),
		SQ.Select("id", "state", "queue", "worker", "args", "meta", "tags", "errors", "attempt", "max_attempts", "priority",
			"inserted_at", "scheduled_at", "attempted_at", "attempted_by", "cancelled_at", "completed_at", "discarded_at").
			From("oban_jobs").
			Where(sq.Eq{"id": jobIDs}))
	if err != nil {
		return err
	}

	// Create a map for fast lookup
	jobMap := make(map[int64]*obanlite.Job)
	for _, job := range jobs {
		jobMap[job.ID] = job
	}

	// Attach jobs to tasks
	for _, t := range tasks {
		if job, ok := jobMap[t.JobID]; ok {
			t.Job = job
		}
	}

	return nil
}

// PreloadTaskSource loads task.source (nil when source_id is nil).
func (st *Store) PreloadTaskSource(ctx context.Context, t *Task) (*Task, error) {
	if t.SourceID == nil {
		t.Source = nil
		return t, nil
	}
	s, err := Get[Source](ctx, st.Q(ctx), *t.SourceID)
	if err != nil {
		return t, err
	}
	t.Source = s
	return t, nil
}

// PreloadTaskMediaItem loads task.media_item (nil when media_item_id is nil).
func (st *Store) PreloadTaskMediaItem(ctx context.Context, t *Task) (*Task, error) {
	if t.MediaItemID == nil {
		t.MediaItem = nil
		return t, nil
	}
	m, err := Get[MediaItem](ctx, st.Q(ctx), *t.MediaItemID)
	if err != nil {
		return t, err
	}
	t.MediaItem = m
	return t, nil
}
