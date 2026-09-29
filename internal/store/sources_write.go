package store

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
)

const (
	sourceUniqueIndex         = "sources_collection_id_media_profile_id_title_filter_regex_index"
	sourceMetadataUniqueIndex = "source_metadata_source_id_index"
)

// CreateSource validates p and inserts a source. Invalid params (including a
// unique violation) come back as a non-empty error map and no error.
func (s *Store) CreateSource(ctx context.Context, p SourceParams) (*Source, map[string][]string, error) {
	base := NewSource()
	p = p.withDefaults(base)
	if errs := p.Validate(base, "pre_insert"); len(errs) > 0 {
		return nil, errs, nil
	}

	rec := p.Apply(base)
	now := db.Now()
	setTimestamp(rec, "inserted_at", now, true)
	setTimestamp(rec, "updated_at", now, true)
	id, err := insertRow(ctx, s.Q(ctx), rec)
	if err != nil {
		if uniqueIndexName(rec.TableName(), err) == sourceUniqueIndex {
			return nil, map[string][]string{"original_url": {"has already been taken"}}, nil
		}
		return nil, nil, err
	}
	rec.ID = id
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
	p = p.withDefaults(source)
	if errs := p.Validate(source, "pre_insert"); len(errs) > 0 {
		return nil, SourceChanges{}, errs, nil
	}

	updated := p.Apply(source)
	changed := p.Changed(source)
	if len(changed) > 0 {
		setTimestamp(updated, "updated_at", db.Now(), false)
		all := fieldsOf(reflect.TypeOf(Source{}))
		set := map[string]any{"updated_at": fieldValue(updated, all["updated_at"])}
		for f := range changed {
			set[f] = fieldValue(updated, all[f])
		}
		if _, err := Exec(ctx, s.Q(ctx), SQ.Update(updated.TableName()).SetMap(set).Where(sq.Eq{"id": updated.ID})); err != nil {
			if uniqueIndexName(updated.TableName(), err) == sourceUniqueIndex {
				return nil, SourceChanges{}, map[string][]string{"original_url": {"has already been taken"}}, nil
			}
			return nil, SourceChanges{}, nil, err
		}
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
	now := db.Now()
	taken := map[string][]string{"metadata.source_id": {"has already been taken"}}
	if rec.ID == 0 {
		setTimestamp(rec, "inserted_at", now, true)
		setTimestamp(rec, "updated_at", now, true)
		id, err := insertRow(ctx, s.Q(ctx), rec)
		if err != nil {
			if uniqueIndexName(rec.TableName(), err) == sourceMetadataUniqueIndex {
				return taken, nil
			}
			return nil, err
		}
		rec.ID = id
	} else {
		setTimestamp(rec, "updated_at", now, false)
		cols, vals := columnValues(rec, false)
		set := map[string]any{}
		for i, c := range cols {
			set[c] = vals[i]
		}
		if _, err := Exec(ctx, s.Q(ctx), SQ.Update(rec.TableName()).SetMap(set).Where(sq.Eq{"id": rec.ID})); err != nil {
			if uniqueIndexName(rec.TableName(), err) == sourceMetadataUniqueIndex {
				return taken, nil
			}
			return nil, err
		}
	}
	parent.Metadata = rec
	return nil, nil
}

func insertRow(ctx context.Context, q db.Querier, rec Schema) (int64, error) {
	cols, vals := columnValues(rec, true)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id", rec.TableName(),
		quoteCols(cols), strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", "))
	var id int64
	err := q.GetContext(ctx, &id, query, vals...)
	return id, err
}

// uniqueIndexName names the unique index a SQLite UNIQUE violation hit, or "".
func uniqueIndexName(table string, err error) string {
	msg := err.Error()
	if m := uniqueIndexRe.FindStringSubmatch(msg); m != nil {
		return m[1]
	}
	if m := uniqueColsRe.FindStringSubmatch(msg); m != nil {
		var cols []string
		for _, c := range strings.Split(m[1], ", ") {
			cols = append(cols, c[strings.IndexByte(c, '.')+1:])
		}
		return table + "_" + strings.Join(cols, "_") + "_index"
	}
	return ""
}
