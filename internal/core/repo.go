package core

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
	t := reflect.TypeOf(zero)
	var out []string
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get("db") == "" {
				walk(f.Type)
				continue
			}
			name := strings.Split(f.Tag.Get("db"), ",")[0]
			if name == "" || name == "-" {
				continue
			}
			if len(alias) > 0 && alias[0] != "" {
				name = alias[0] + "." + name + ` AS "` + name + `"`
			}
			out = append(out, name)
		}
	}
	walk(t)
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

// Insert validates and inserts a changeset over *T (Repo.insert/1). All
// columns are written (like Ecto), timestamps set, and any cast has_one
// assoc inserted after. Invalid changesets and mapped unique violations
// return *ChangesetError with Action "insert".
func Insert[T Schema](ctx context.Context, q db.Querier, cs *Changeset) (*T, error) {
	cs.Action = "insert"
	if !cs.Valid() {
		return nil, &ChangesetError{cs}
	}
	rec := cs.Apply().(*T)
	now := db.Now()
	setTimestamp(rec, "inserted_at", now, true)
	setTimestamp(rec, "updated_at", now, true)

	cols, vals := columnValues(rec, true)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id", (*rec).TableName(),
		quoteCols(cols), strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", "))
	var id int64
	if err := q.GetContext(ctx, &id, query, vals...); err != nil {
		return nil, cs.mapConstraintError((*rec).TableName(), err)
	}
	setID(rec, id)
	if err := saveAssocs(ctx, q, cs, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// Update applies a changeset (Repo.update/1): only changed columns plus
// updated_at are written; with no changes nothing is written.
func Update[T Schema](ctx context.Context, q db.Querier, cs *Changeset) (*T, error) {
	cs.Action = "update"
	if !cs.Valid() {
		return nil, &ChangesetError{cs}
	}
	rec := cs.Apply().(*T)
	if len(cs.Changes) > 0 {
		now := db.Now()
		setTimestamp(rec, "updated_at", now, false)
		set := map[string]any{}
		all := fieldsOf(reflect.TypeOf(rec).Elem())
		for field := range cs.Changes {
			set[field] = fieldValue(rec, all[field])
		}
		if _, ok := all["updated_at"]; ok {
			set["updated_at"] = fieldValue(rec, all["updated_at"])
		}
		_, err := Exec(ctx, q, SQ.Update((*rec).TableName()).SetMap(set).Where(sq.Eq{"id": idOf(rec)}))
		if err != nil {
			return nil, cs.mapConstraintError((*rec).TableName(), err)
		}
	}
	if err := saveAssocs(ctx, q, cs, rec); err != nil {
		return nil, err
	}
	return rec, nil
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

func saveAssocs(ctx context.Context, q db.Querier, cs *Changeset, parent any) error {
	if len(cs.assocs) == 0 {
		return nil
	}
	pid := idOf(parent)
	fk := strings.TrimSuffix(parent.(Schema).TableName(), "s") + "_id"
	for name, child := range cs.assocs {
		if !child.Valid() {
			return &ChangesetError{cs}
		}
		child.PutChange(fk, pid)
		rec := child.Apply()
		table := rec.(Schema).TableName()
		now := db.Now()
		if idOf(rec) == 0 {
			setTimestamp(rec, "inserted_at", now, true)
			setTimestamp(rec, "updated_at", now, true)
			cols, vals := columnValues(rec, true)
			query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id", table, quoteCols(cols),
				strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", "))
			var id int64
			if err := q.GetContext(ctx, &id, query, vals...); err != nil {
				return child.mapConstraintError(table, err)
			}
			setID(rec, id)
		} else if len(child.Changes) > 0 {
			setTimestamp(rec, "updated_at", now, false)
			cols, vals := columnValues(rec, false)
			set := map[string]any{}
			for i, c := range cols {
				set[c] = vals[i]
			}
			if _, err := Exec(ctx, q, SQ.Update(table).SetMap(set).Where(sq.Eq{"id": idOf(rec)})); err != nil {
				return child.mapConstraintError(table, err)
			}
		}
		// Point the parent's association field at the saved child.
		pv := reflect.ValueOf(parent).Elem()
		for i := 0; i < pv.NumField(); i++ {
			f := pv.Type().Field(i)
			if strings.EqualFold(f.Name, snakeToCamel(name)) && f.Type == reflect.TypeOf(rec) {
				pv.Field(i).Set(reflect.ValueOf(rec))
			}
		}
	}
	return nil
}

var (
	uniqueColsRe  = regexp.MustCompile(`UNIQUE constraint failed: ([\w.]+(?:, [\w.]+)*)`)
	uniqueIndexRe = regexp.MustCompile(`UNIQUE constraint failed: index '(\w+)'`)
)

// mapConstraintError turns a UNIQUE violation into a changeset error when a
// matching UniqueConstraint was declared, like Ecto. Otherwise the raw
// error is returned (Ecto would raise).
func (cs *Changeset) mapConstraintError(table string, err error) error {
	msg := err.Error()
	var name string
	if m := uniqueIndexRe.FindStringSubmatch(msg); m != nil {
		name = m[1]
	} else if m := uniqueColsRe.FindStringSubmatch(msg); m != nil {
		var cols []string
		for _, c := range strings.Split(m[1], ", ") {
			cols = append(cols, c[strings.IndexByte(c, '.')+1:])
		}
		name = table + "_" + strings.Join(cols, "_") + "_index"
	} else {
		return err
	}
	for _, u := range cs.uniques {
		if table+"_"+strings.Join(u.columns, "_")+"_index" == name {
			cs.AddError(u.field, u.message, map[string]any{"constraint": "unique", "constraint_name": name})
			return &ChangesetError{cs}
		}
	}
	return err
}

func columnValues(rec any, skipZeroID bool) ([]string, []any) {
	fields := fieldsOf(reflect.TypeOf(rec).Elem())
	var cols []string
	var vals []any
	for _, name := range orderedColumns(reflect.TypeOf(rec).Elem()) {
		fi := fields[name]
		if name == "id" && skipZeroID && idOf(rec) == 0 {
			continue
		}
		cols = append(cols, name)
		vals = append(vals, fieldValue(rec, fi))
	}
	return cols, vals
}

func orderedColumns(t reflect.Type) []string {
	var out []string
	var walk func(t reflect.Type)
	walk = func(t reflect.Type) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if f.Anonymous && f.Type.Kind() == reflect.Struct && f.Tag.Get("db") == "" {
				walk(f.Type)
				continue
			}
			name := strings.Split(f.Tag.Get("db"), ",")[0]
			if name != "" && name != "-" {
				out = append(out, name)
			}
		}
	}
	walk(t)
	return out
}

func fieldValue(rec any, fi fieldInfo) any {
	fv := reflect.ValueOf(rec).Elem().FieldByIndex(fi.index)
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

func setTimestamp(rec any, col string, now db.UTCDateTime, onlyIfZero bool) {
	fi, ok := fieldsOf(reflect.TypeOf(rec).Elem())[col]
	if !ok {
		return
	}
	f := reflect.ValueOf(rec).Elem().FieldByIndex(fi.index)
	if onlyIfZero {
		if t, ok := f.Interface().(db.UTCDateTime); ok && !t.IsZero() {
			return
		}
	}
	setField(f, now)
}

func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
