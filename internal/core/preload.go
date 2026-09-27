package core

// Association loading: the Repo.preload/2 calls Pinchflat makes. Each helper
// always (re)loads, like `force: true`, and returns the same pointer for
// chaining. Hand-written W0 infrastructure.

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// PreloadSourceMediaProfile loads source.media_profile.
func (a *App) PreloadSourceMediaProfile(ctx context.Context, s *Source) (*Source, error) {
	p, err := Get[MediaProfile](ctx, a.Q(ctx), s.MediaProfileID)
	if err != nil {
		return s, err
	}
	s.MediaProfile = p
	return s, nil
}

// PreloadSourceMetadata loads source.metadata (nil if none).
func (a *App) PreloadSourceMetadata(ctx context.Context, s *Source) (*Source, error) {
	m, err := One[SourceMetadata](ctx, a.Q(ctx), From[SourceMetadata]().Where(sq.Eq{"source_id": s.ID}))
	if err != nil {
		return s, err
	}
	s.Metadata = m
	return s, nil
}

// PreloadSourceMediaItems loads source.media_items ordered by id.
func (a *App) PreloadSourceMediaItems(ctx context.Context, s *Source) (*Source, error) {
	items, err := All[MediaItem](ctx, a.Q(ctx), From[MediaItem]().Where(sq.Eq{"source_id": s.ID}).OrderBy("id"))
	if err != nil {
		return s, err
	}
	s.MediaItems = items
	return s, nil
}

// PreloadSourceTasks loads source.tasks with each task's job
// (preload(tasks: [:job])).
func (a *App) PreloadSourceTasks(ctx context.Context, s *Source) (*Source, error) {
	tasks, err := All[Task](ctx, a.Q(ctx), From[Task]().Where(sq.Eq{"source_id": s.ID}).OrderBy("id"))
	if err != nil {
		return s, err
	}
	for _, t := range tasks {
		if _, err := a.PreloadTaskJob(ctx, t); err != nil {
			return s, err
		}
	}
	s.Tasks = tasks
	return s, nil
}

// PreloadMediaItemSource loads media_item.source.
func (a *App) PreloadMediaItemSource(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	s, err := Get[Source](ctx, a.Q(ctx), m.SourceID)
	if err != nil {
		return m, err
	}
	m.Source = s
	return m, nil
}

// PreloadMediaItemSourceAndProfile loads media_item.source.media_profile
// (preload(source: :media_profile)).
func (a *App) PreloadMediaItemSourceAndProfile(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	if _, err := a.PreloadMediaItemSource(ctx, m); err != nil {
		return m, err
	}
	_, err := a.PreloadSourceMediaProfile(ctx, m.Source)
	return m, err
}

// PreloadMediaItemMetadata loads media_item.metadata (nil if none).
func (a *App) PreloadMediaItemMetadata(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	md, err := One[MediaMetadata](ctx, a.Q(ctx), From[MediaMetadata]().Where(sq.Eq{"media_item_id": m.ID}))
	if err != nil {
		return m, err
	}
	m.Metadata = md
	return m, nil
}

// PreloadMediaItemTasks loads media_item.tasks with their jobs.
func (a *App) PreloadMediaItemTasks(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	tasks, err := All[Task](ctx, a.Q(ctx), From[Task]().Where(sq.Eq{"media_item_id": m.ID}).OrderBy("id"))
	if err != nil {
		return m, err
	}
	for _, t := range tasks {
		if _, err := a.PreloadTaskJob(ctx, t); err != nil {
			return m, err
		}
	}
	m.Tasks = tasks
	return m, nil
}

// PreloadMediaItemFull loads [:metadata, source: :media_profile].
func (a *App) PreloadMediaItemFull(ctx context.Context, m *MediaItem) (*MediaItem, error) {
	if _, err := a.PreloadMediaItemMetadata(ctx, m); err != nil {
		return m, err
	}
	return a.PreloadMediaItemSourceAndProfile(ctx, m)
}

// PreloadMediaProfileSources loads media_profile.sources.
func (a *App) PreloadMediaProfileSources(ctx context.Context, p *MediaProfile) (*MediaProfile, error) {
	sources, err := All[Source](ctx, a.Q(ctx), From[Source]().Where(sq.Eq{"media_profile_id": p.ID}).OrderBy("id"))
	if err != nil {
		return p, err
	}
	p.Sources = sources
	return p, nil
}

// PreloadTaskJob loads task.job.
func (a *App) PreloadTaskJob(ctx context.Context, t *Task) (*Task, error) {
	var job obanlite.Job
	err := a.Q(ctx).GetContext(ctx, &job, `SELECT id, state, queue, worker, args, meta, tags, errors, attempt, max_attempts, priority,
		inserted_at, scheduled_at, attempted_at, attempted_by, cancelled_at, completed_at, discarded_at FROM oban_jobs WHERE id = ?`, t.JobID)
	if err != nil {
		return t, err
	}
	t.Job = &job
	return t, nil
}

// PreloadTaskSource loads task.source (nil when source_id is nil).
func (a *App) PreloadTaskSource(ctx context.Context, t *Task) (*Task, error) {
	if t.SourceID == nil {
		t.Source = nil
		return t, nil
	}
	s, err := Get[Source](ctx, a.Q(ctx), *t.SourceID)
	if err != nil {
		return t, err
	}
	t.Source = s
	return t, nil
}

// PreloadTaskMediaItem loads task.media_item (nil when media_item_id is nil).
func (a *App) PreloadTaskMediaItem(ctx context.Context, t *Task) (*Task, error) {
	if t.MediaItemID == nil {
		t.MediaItem = nil
		return t, nil
	}
	m, err := Get[MediaItem](ctx, a.Q(ctx), *t.MediaItemID)
	if err != nil {
		return t, err
	}
	t.MediaItem = m
	return t, nil
}
