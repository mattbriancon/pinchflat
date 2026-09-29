package store

// Repo helpers: the Ecto.Repo operations Pinchflat uses, over db-tagged
// schema structs. Hand-written W0 infrastructure; not a manifest row.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
)

// Schema is implemented by every schema struct (value receiver).
type Schema interface{ TableName() string }

// ErrNotFound is returned where Ecto would raise Ecto.NoResultsError
// (Repo.get!/2, Repo.one!/1).
var ErrNotFound = errors.New("not found")

// SQ is the squirrel builder configured for SQLite placeholders.
var SQ = sq.StatementBuilder.PlaceholderFormat(sq.Question)

// Columns returns the db columns of schema T in field order, for SELECTs.
// Qualify with a table alias by passing it (e.g. Columns[Source]("s")).
func Columns[T Schema](alias ...string) []string {
	var zero T
	var out []string
	for _, c := range columnsOf(reflect.TypeOf(zero)) {
		name := c.name
		if len(alias) > 0 && alias[0] != "" {
			name = alias[0] + "." + name + ` AS "` + name + `"`
		}
		out = append(out, name)
	}
	return out
}

// From starts a SELECT of every column of T from its table.
func From[T Schema](alias ...string) sq.SelectBuilder {
	var zero T
	table := zero.TableName()
	if len(alias) > 0 && alias[0] != "" {
		return SQ.Select(Columns[T](alias[0])...).From(table + " AS " + alias[0])
	}
	return SQ.Select(Columns[T]()...).From(table)
}

// All runs a query built with squirrel and scans every row into T.
func All[T any](ctx context.Context, q db.Querier, query sq.Sqlizer) ([]*T, error) {
	s, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	var out []*T
	if err := q.SelectContext(ctx, &out, s, args...); err != nil {
		return nil, fmt.Errorf("%w\n  query: %s", err, s)
	}
	return out, nil
}

// One runs a query expected to return at most one row; (nil, nil) if none
// (Repo.one/1).
func One[T any](ctx context.Context, q db.Querier, query sq.Sqlizer) (*T, error) {
	s, args, err := query.ToSql()
	if err != nil {
		return nil, err
	}
	var out T
	if err := q.GetContext(ctx, &out, s, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w\n  query: %s", err, s)
	}
	return &out, nil
}

// MustOne is One returning ErrNotFound when there's no row (Repo.one!/1).
func MustOne[T any](ctx context.Context, q db.Querier, query sq.Sqlizer) (*T, error) {
	r, err := One[T](ctx, q, query)
	if err == nil && r == nil {
		return nil, ErrNotFound
	}
	return r, err
}

// Scalar runs a query returning a single value (count, exists, ...).
func Scalar[T any](ctx context.Context, q db.Querier, query sq.Sqlizer) (T, error) {
	var out T
	s, args, err := query.ToSql()
	if err != nil {
		return out, err
	}
	err = q.GetContext(ctx, &out, s, args...)
	return out, err
}

// Get loads T by primary key (Repo.get!/2), returning ErrNotFound.
func Get[T Schema](ctx context.Context, q db.Querier, id int64) (*T, error) {
	var zero T
	return MustOne[T](ctx, q, From[T]().Where(sq.Eq{zero.TableName() + ".id": id}))
}

// Reload re-reads a record (Repo.reload!/1). Associations are not loaded.
func Reload[T Schema](ctx context.Context, q db.Querier, rec *T) (*T, error) {
	return Get[T](ctx, q, idOf(rec))
}

// Exec runs a squirrel INSERT/UPDATE/DELETE and returns rows affected.
func Exec(ctx context.Context, q db.Querier, query sq.Sqlizer) (int64, error) {
	s, args, err := query.ToSql()
	if err != nil {
		return 0, err
	}
	res, err := q.ExecContext(ctx, s, args...)
	if err != nil {
		return 0, fmt.Errorf("%w\n  query: %s", err, s)
	}
	return res.RowsAffected()
}

// Insert stamps rec's inserted_at/updated_at (unless already set), writes
// every column (like Ecto) and sets rec.ID. A UNIQUE violation on a known
// index comes back as ValidationErrors.
func Insert[T Schema](ctx context.Context, q db.Querier, rec *T) error {
	return insertRow(ctx, q, rec, "")
}

// insertRow is Insert with an optional trailing clause (ON CONFLICT ...) and
// its arguments.
func insertRow[T Schema](ctx context.Context, q db.Querier, rec *T, suffix string, suffixArgs ...any) error {
	now := db.Now()
	setTimestamp(rec, "inserted_at", now, true)
	setTimestamp(rec, "updated_at", now, true)

	cols, vals := columnValues(rec, true)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)%s RETURNING id", (*rec).TableName(),
		quoteCols(cols), strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", "), suffix)
	var id int64
	if err := q.GetContext(ctx, &id, query, append(vals, suffixArgs...)...); err != nil {
		return uniqueViolation((*rec).TableName(), err)
	}
	setID(rec, id)
	return nil
}

// Update writes the named columns of rec plus updated_at (Repo.update/1);
// with no columns nothing is written. A UNIQUE violation on a known index
// comes back as ValidationErrors.
func Update[T Schema](ctx context.Context, q db.Querier, rec *T, cols ...string) error {
	if len(cols) == 0 {
		return nil
	}
	set := map[string]any{}
	for _, c := range cols {
		set[c] = columnValue(rec, c)
	}
	if hasColumn(rec, "updated_at") {
		setTimestamp(rec, "updated_at", db.Now(), false)
		set["updated_at"] = columnValue(rec, "updated_at")
	}
	if _, err := Exec(ctx, q, SQ.Update((*rec).TableName()).SetMap(set).Where(sq.Eq{"id": idOf(rec)})); err != nil {
		return uniqueViolation((*rec).TableName(), err)
	}
	return nil
}

// Delete deletes a record by id (Repo.delete/1).
func Delete[T Schema](ctx context.Context, q db.Querier, rec *T) error {
	n, err := Exec(ctx, q, SQ.Delete((*rec).TableName()).Where(sq.Eq{"id": idOf(rec)}))
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound // Ecto.StaleEntryError
	}
	return nil
}

var (
	uniqueColsRe  = regexp.MustCompile(`UNIQUE constraint failed: ([\w.]+(?:, [\w.]+)*)`)
	uniqueIndexRe = regexp.MustCompile(`UNIQUE constraint failed: index '(\w+)'`)
)

// uniqueErrorFields maps each unique index the app reports to the field its
// "has already been taken" error is keyed under.
var uniqueErrorFields = map[string]string{
	"sources_collection_id_media_profile_id_title_filter_regex_index": "original_url",
	"source_metadata_source_id_index":                                 "metadata.source_id",
	"media_items_media_id_source_id_index":                            "media_id",
	"media_metadata_media_item_id_index":                              "metadata.media_item_id",
	"media_profiles_name_index":                                       "name",
}

// uniqueViolation turns a SQLite UNIQUE violation on one of
// uniqueErrorFields' indexes into ValidationErrors (Ecto's unique_constraint).
// Any other error is returned as is (Ecto would raise).
func uniqueViolation(table string, err error) error {
	msg := err.Error()
	var index string
	if m := uniqueIndexRe.FindStringSubmatch(msg); m != nil {
		index = m[1]
	} else if m := uniqueColsRe.FindStringSubmatch(msg); m != nil {
		var cols []string
		for _, c := range strings.Split(m[1], ", ") {
			cols = append(cols, c[strings.IndexByte(c, '.')+1:])
		}
		index = table + "_" + strings.Join(cols, "_") + "_index"
	}
	if field, ok := uniqueErrorFields[index]; ok {
		return ValidationErrors{field: {"has already been taken"}}
	}
	return err
}

// column is a db column of a schema struct and the index of its field.
type column struct {
	name  string
	index []int
}

var columnCache sync.Map // reflect.Type -> []column

// columnsOf lists t's db columns in field order. Fields tagged db:"-", virtual
// or without a db tag are skipped; untagged embedded structs are flattened.
func columnsOf(t reflect.Type) []column {
	if c, ok := columnCache.Load(t); ok {
		return c.([]column)
	}
	var out []column
	var walk func(t reflect.Type, prefix []int)
	walk = func(t reflect.Type, prefix []int) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			idx := append(append([]int{}, prefix...), i)
			if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get("db") == "" {
				walk(f.Type, idx)
				continue
			}
			name := strings.Split(f.Tag.Get("db"), ",")[0]
			if name == "" || name == "-" || isVirtual(f.Tag.Get("db")) {
				continue
			}
			out = append(out, column{name, idx})
		}
	}
	walk(t, nil)
	columnCache.Store(t, out)
	return out
}

func columnValues(rec any, skipZeroID bool) ([]string, []any) {
	var cols []string
	var vals []any
	rv := reflect.ValueOf(rec).Elem()
	for _, c := range columnsOf(rv.Type()) {
		if c.name == "id" && skipZeroID && idOf(rec) == 0 {
			continue
		}
		cols = append(cols, c.name)
		vals = append(vals, fieldValue(rv.FieldByIndex(c.index)))
	}
	return cols, vals
}

func hasColumn(rec any, name string) bool {
	for _, c := range columnsOf(reflect.TypeOf(rec).Elem()) {
		if c.name == name {
			return true
		}
	}
	return false
}

// columnValue is rec's value for one column.
func columnValue(rec any, name string) any {
	rv := reflect.ValueOf(rec).Elem()
	for _, c := range columnsOf(rv.Type()) {
		if c.name == name {
			return fieldValue(rv.FieldByIndex(c.index))
		}
	}
	panic(fmt.Sprintf("%T has no column %q", rec, name))
}

// fieldValue is the value to write for a field: nil pointers come back as
// untyped nil (NULL).
func fieldValue(fv reflect.Value) any {
	if fv.Kind() == reflect.Pointer && fv.IsNil() {
		return nil
	}
	return fv.Interface()
}

func quoteCols(cols []string) string {
	q := make([]string, len(cols))
	for i, c := range cols {
		q[i] = `"` + c + `"`
	}
	return strings.Join(q, ", ")
}

func idOf(rec any) int64 {
	f := reflect.ValueOf(rec).Elem().FieldByName("ID")
	if !f.IsValid() {
		return 0
	}
	return f.Int()
}

func setID(rec any, id int64) {
	reflect.ValueOf(rec).Elem().FieldByName("ID").SetInt(id)
}

// setTimestamp stamps a db.UTCDateTime column with now (only when it is still
// zero, if onlyIfZero).
func setTimestamp(rec any, col string, now db.UTCDateTime, onlyIfZero bool) {
	rv := reflect.ValueOf(rec).Elem()
	for _, c := range columnsOf(rv.Type()) {
		if c.name != col {
			continue
		}
		f := rv.FieldByIndex(c.index)
		if t, ok := f.Interface().(db.UTCDateTime); ok && onlyIfZero && !t.IsZero() {
			return
		}
		f.Set(reflect.ValueOf(now))
	}
}

// isVirtual reports a `db:"name,virtual"` tag: a field that queries may
// select into (e.g. media_items.matching_search_term) but that is not a
// table column, so it's excluded from Columns, inserts and updates.
func isVirtual(tag string) bool {
	for _, opt := range strings.Split(tag, ",")[1:] {
		if opt == "virtual" {
			return true
		}
	}
	return false
}
