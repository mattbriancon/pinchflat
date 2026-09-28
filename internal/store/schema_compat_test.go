package store_test

// Hand-written W0 compatibility test: every schema struct must cover every
// column of its table and round-trip Elixir-written rows byte for byte.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

func tableColumns(t *testing.T, d *db.DB, table string) []string {
	t.Helper()
	var cols []string
	if err := d.Select(&cols, `SELECT name FROM pragma_table_info(?)`, table); err != nil {
		t.Fatal(err)
	}
	sort.Strings(cols)
	return cols
}

// rawRows returns quote(col) for every column of every row, keyed by id.
func rawRows(t *testing.T, d *db.DB, table string, cols []string) map[int64]string {
	t.Helper()
	var exprs []string
	for _, c := range cols {
		exprs = append(exprs, fmt.Sprintf(`quote("%s")`, c))
	}
	rows, err := d.Query(`SELECT id, ` + strings.Join(exprs, " || '|' || ") + ` FROM ` + table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var s string
		if err := rows.Scan(&id, &s); err != nil {
			t.Fatal(err)
		}
		out[id] = s
	}
	return out
}

// retiredColumns are kept in the database (never dropped, so the schema
// stays what Elixir left) but no longer used by the Go app.
var retiredColumns = map[string][]string{
	"settings": {"apprise_server", "apprise_version", "pro_enabled"}, // Apprise and Pro mode removed
}

func withoutRetired(table string, cols []string) []string {
	var out []string
outer:
	for _, c := range cols {
		for _, r := range retiredColumns[table] {
			if c == r {
				continue outer
			}
		}
		out = append(out, c)
	}
	return out
}

func checkSchema[T store.Schema](t *testing.T) {
	var zero T
	table := zero.TableName()
	t.Run(table, func(t *testing.T) {
		ctx := context.Background()
		d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))

		dbCols := tableColumns(t, d, table)
		want := withoutRetired(table, dbCols)
		got := store.Columns[T]()
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("%T columns\n got: %v\nwant: %v", zero, got, want)
		}

		// Compare every database column, so retired ones must survive too.
		before := rawRows(t, d, table, dbCols)
		recs, err := store.All[T](ctx, d, store.From[T]())
		if err != nil {
			t.Fatalf("scan %s: %v", table, err)
		}
		if len(recs) == 0 {
			t.Fatalf("fixture has no %s rows", table)
		}

		// Write every column back from the struct; nothing may change,
		// except legacy uploaded_at values, which are rewritten in
		// Ecto's current format.
		for _, rec := range recs {
			cs := store.Change(rec, nil)
			for _, c := range want {
				cs.Changes[c] = cs.GetField(c)
			}
			set := map[string]any{}
			for c, v := range cs.Changes {
				set[c] = v
			}
			id := cs.GetField("id")
			if _, err := store.Exec(ctx, d, store.SQ.Update(table).SetMap(set).Where(sq.Eq{"id": id})); err != nil {
				t.Fatalf("write back %s %v: %v", table, id, err)
			}
		}
		after := rawRows(t, d, table, dbCols)
		for id, b := range before {
			if after[id] == b {
				continue
			}
			if table == "media_items" && legacyUploadedAtOnly(dbCols, b, after[id]) {
				continue
			}
			t.Errorf("%s row %d changed on round trip\n before: %s\n  after: %s", table, id, b, after[id])
		}
	})
}

func legacyUploadedAtOnly(cols []string, before, after string) bool {
	b, a := strings.Split(before, "|"), strings.Split(after, "|")
	for i, c := range cols {
		if b[i] != a[i] && c != "uploaded_at" {
			return false
		}
	}
	return true
}

func TestSchemasMatchElixirTables(t *testing.T) {
	checkSchema[store.MediaProfile](t)
	checkSchema[store.Source](t)
	checkSchema[store.MediaItem](t)
	checkSchema[store.MediaMetadata](t)
	checkSchema[store.SourceMetadata](t)
	checkSchema[store.Setting](t)
	checkSchema[store.Task](t)
}
