package store

import (
	"context"

	sq "github.com/Masterminds/squirrel"
)

// UseCookies reports whether cookies should be used for operation.
func UseCookies(source *Source, operation string) bool {
	switch source.CookieBehaviour {
	case SourceCookieBehaviourDisabled:
		return false
	case SourceCookieBehaviourAllOperations:
		return true
	case SourceCookieBehaviourWhenNeeded:
		return operation == "indexing" || operation == "error_recovery"
	}
	return false
}

// ListSources returns every source.
func (s *Store) ListSources(ctx context.Context) ([]*Source, error) {
	return All[Source](ctx, s.Q(ctx), From[Source]())
}

// ListSourcesFor returns the sources using profile.
func (s *Store) ListSourcesFor(ctx context.Context, profile *MediaProfile) ([]*Source, error) {
	return All[Source](ctx, s.Q(ctx), From[Source]().Where(sq.Eq{"media_profile_id": profile.ID}))
}

// GetSource returns ErrNotFound if the source doesn't exist.
func (s *Store) GetSource(ctx context.Context, id int64) (*Source, error) {
	return Get[Source](ctx, s.Q(ctx), id)
}
