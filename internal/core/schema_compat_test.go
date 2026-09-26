package core_test

// Hand-written W0 compatibility test: every schema struct must cover every
// column of its table and round-trip Elixir-written rows byte for byte.

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
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

func checkSchema[T core.Schema](t *testing.T) {
	var zero T
	table := zero.TableName()
	t.Run(table, func(t *testing.T) {
		ctx := context.Background()
		d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))

		want := tableColumns(t, d, table)
		got := core.Columns[T]()
		sort.Strings(got)
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("%T columns\n got: %v\nwant: %v", zero, got, want)
		}

		before := rawRows(t, d, table, want)
		recs, err := core.All[T](ctx, d, core.From[T]())
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
			cs := core.Change(rec, nil)
			for _, c := range want {
				cs.Changes[c] = cs.GetField(c)
			}
			set := map[string]any{}
			for c, v := range cs.Changes {
				set[c] = v
			}
			id := cs.GetField("id")
			if _, err := core.Exec(ctx, d, core.SQ.Update(table).SetMap(set).Where(sq.Eq{"id": id})); err != nil {
				t.Fatalf("write back %s %v: %v", table, id, err)
			}
		}
		after := rawRows(t, d, table, want)
		for id, b := range before {
			if after[id] == b {
				continue
			}
			if table == "media_items" && legacyUploadedAtOnly(want, b, after[id]) {
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
	checkSchema[core.MediaProfile](t)
	checkSchema[core.Source](t)
	checkSchema[core.MediaItem](t)
	checkSchema[core.MediaMetadata](t)
	checkSchema[core.SourceMetadata](t)
	checkSchema[core.Setting](t)
	checkSchema[core.Task](t)
}
