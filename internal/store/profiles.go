package store

import (
	"context"

	sq "github.com/Masterminds/squirrel"
)

// ListMediaProfiles returns every media profile.
func (s *Store) ListMediaProfiles(ctx context.Context) ([]*MediaProfile, error) {
	return All[MediaProfile](ctx, s.Q(ctx), From[MediaProfile](""))
}

// GetMediaProfile returns ErrNotFound if the profile doesn't exist.
func (s *Store) GetMediaProfile(ctx context.Context, id int64) (*MediaProfile, error) {
	return MustOne[MediaProfile](ctx, s.Q(ctx), From[MediaProfile]("").Where(sq.Eq{"id": id}))
}

// CreateMediaProfile creates a media profile from attrs.
func (s *Store) CreateMediaProfile(ctx context.Context, attrs Attrs) (*MediaProfile, error) {
	profile := NewMediaProfile()
	cs := MediaProfileChangeset(profile, attrs)
	return Insert[MediaProfile](ctx, s.Q(ctx), cs)
}

// UpdateMediaProfile updates profile with attrs.
func (s *Store) UpdateMediaProfile(ctx context.Context, profile *MediaProfile, attrs Attrs) (*MediaProfile, error) {
	cs := MediaProfileChangeset(profile, attrs)
	return Update[MediaProfile](ctx, s.Q(ctx), cs)
}

// ChangeMediaProfile builds a changeset for profile from attrs.
func (s *Store) ChangeMediaProfile(ctx context.Context, profile *MediaProfile, attrs Attrs) *Changeset {
	return MediaProfileChangeset(profile, attrs)
}
