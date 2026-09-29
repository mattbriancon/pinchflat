package store

import (
	"context"
)

// CreateSource validates p and inserts a source. Invalid params (including a
// unique violation) come back as a non-empty error map and no error.
func (s *Store) CreateSource(ctx context.Context, p SourceParams) (*Source, map[string][]string, error) {
	base := NewSource()
	if errs := p.Validate(base, "pre_insert"); len(errs) > 0 {
		return nil, errs, nil
	}

	rec, _ := p.apply(base)
	if err := Insert(ctx, s.Q(ctx), rec); err != nil {
		if errs, ok := AsValidationErrors(err); ok {
			return nil, errs, nil
		}
		return nil, nil, err
	}
	if p.Metadata != nil {
		if errs, err := s.saveSourceMetadata(ctx, rec, p.Metadata, nil); err != nil || errs != nil {
			return nil, errs, err
		}
	}
	return rec, nil, nil
}

// UpdateSource validates p against source and writes the changed columns.
// Invalid params come back as a non-empty error map and no error.
func (s *Store) UpdateSource(ctx context.Context, source *Source, p SourceParams) (*Source, SourceChanges, map[string][]string, error) {
	if errs := p.Validate(source, "pre_insert"); len(errs) > 0 {
		return nil, SourceChanges{}, errs, nil
	}

	updated, changed := p.apply(source)
	if err := Update(ctx, s.Q(ctx), updated, changed.list()...); err != nil {
		if errs, ok := AsValidationErrors(err); ok {
			return nil, SourceChanges{}, errs, nil
		}
		return nil, SourceChanges{}, nil, err
	}
	if p.Metadata != nil {
		if errs, err := s.saveSourceMetadata(ctx, updated, p.Metadata, source.Metadata); err != nil || errs != nil {
			return nil, SourceChanges{}, errs, err
		}
	}
	return updated, SourceChanges{Changed: changed, Before: source, After: updated}, nil, nil
}

// saveSourceMetadata inserts (current == nil) or rewrites the source's
// source_metadata row and points parent.Metadata at it.
func (s *Store) saveSourceMetadata(ctx context.Context, parent *Source, m *SourceMetadata, current *SourceMetadata) (map[string][]string, error) {
	rec := applySourceMetadata(current, m)
	rec.SourceID = parent.ID
	var err error
	if rec.ID == 0 {
		err = Insert(ctx, s.Q(ctx), rec)
	} else {
		err = Update(ctx, s.Q(ctx), rec, "metadata_filepath", "fanart_filepath", "poster_filepath", "banner_filepath", "source_id")
	}
	if err != nil {
		if errs, ok := AsValidationErrors(err); ok {
			return errs, nil
		}
		return nil, err
	}
	parent.Metadata = rec
	return nil, nil
}
