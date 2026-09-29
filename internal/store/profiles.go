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

// CreateMediaProfile creates a media profile from p. When p is invalid it
// returns the validation errors and no profile.
func (s *Store) CreateMediaProfile(ctx context.Context, p MediaProfileParams) (*MediaProfile, map[string][]string, error) {
	base := NewMediaProfile()
	if errs := p.Validate(base); len(errs) > 0 {
		return nil, errs, nil
	}
	rec := p.Apply(base)
	if err := Insert(ctx, s.Q(ctx), rec); err != nil {
		if errs, ok := AsValidationErrors(err); ok {
			return nil, errs, nil
		}
		return nil, nil, err
	}
	return rec, nil, nil
}

// UpdateMediaProfile applies p to profile. When p is invalid it returns the
// validation errors and writes nothing.
func (s *Store) UpdateMediaProfile(ctx context.Context, profile *MediaProfile, p MediaProfileParams) (*MediaProfile, map[string][]string, error) {
	if errs := p.Validate(profile); len(errs) > 0 {
		return nil, errs, nil
	}
	rec, changed := p.apply(profile)
	if len(changed) == 0 {
		return rec, nil, nil
	}
	if err := Update(ctx, s.Q(ctx), rec, changed...); err != nil {
		if errs, ok := AsValidationErrors(err); ok {
			return nil, errs, nil
		}
		return nil, nil, err
	}
	return rec, nil, nil
}
