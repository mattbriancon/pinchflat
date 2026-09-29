package store

import (
	"context"
	"fmt"
	"slices"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
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
	now := db.Now()
	setTimestamp(rec, "inserted_at", now, true)
	setTimestamp(rec, "updated_at", now, true)

	cols, vals := columnValues(rec, true)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id", rec.TableName(),
		quoteCols(cols), strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", "))
	var id int64
	if err := s.Q(ctx).GetContext(ctx, &id, query, vals...); err != nil {
		if isMediaProfileNameTaken(err) {
			return nil, map[string][]string{"name": {"has already been taken"}}, nil
		}
		return nil, nil, err
	}
	setID(rec, id)
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
	setTimestamp(rec, "updated_at", db.Now(), false)
	cols, vals := columnValues(rec, false)
	set := map[string]any{}
	for i, c := range cols {
		if c == "updated_at" || slices.Contains(changed, c) {
			set[c] = vals[i]
		}
	}
	if _, err := Exec(ctx, s.Q(ctx), SQ.Update(rec.TableName()).SetMap(set).Where(sq.Eq{"id": rec.ID})); err != nil {
		if isMediaProfileNameTaken(err) {
			return nil, map[string][]string{"name": {"has already been taken"}}, nil
		}
		return nil, nil, err
	}
	return rec, nil, nil
}

func isMediaProfileNameTaken(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed: media_profiles.name") ||
		strings.Contains(msg, "UNIQUE constraint failed: index 'media_profiles_name_index'")
}
