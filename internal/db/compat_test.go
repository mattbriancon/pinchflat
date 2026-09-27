package db_test

import (
	"bufio"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
)

type masterRow struct {
	Type    string `db:"type"`
	Name    string `db:"name"`
	TblName string `db:"tbl_name"`
	SQL     string `db:"sql"`
}

// schema returns sqlite_master minus things that aren't part of the app
// schema: sqlean_define is created by the sqlean extension on load, and
// autoindexes have no SQL.
func schema(t *testing.T, d *db.DB) map[string]masterRow {
	t.Helper()
	var rows []masterRow
	err := d.Select(&rows, `SELECT type, name, tbl_name, COALESCE(sql, '') AS sql FROM sqlite_master
		WHERE name NOT IN ('sqlean_define') AND name NOT LIKE 'sqlite_autoindex%'`)
	if err != nil {
		t.Fatal(err)
	}
	m := map[string]masterRow{}
	for _, r := range rows {
		m[r.Type+" "+r.Name] = r
	}
	return m
}

func assertSameSchema(t *testing.T, want, got map[string]masterRow) {
	t.Helper()
	for k, w := range want {
		g, ok := got[k]
		if !ok {
			t.Errorf("missing %s", k)
			continue
		}
		if w != g {
			t.Errorf("%s differs:\n want: %s\n  got: %s", k, w.SQL, g.SQL)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("unexpected %s", k)
		}
	}
}

func TestFreshMigrationMatchesElixirSchema(t *testing.T) {
	want := schema(t, dbtest.CopyOf(t, dbtest.ElixirFixture("schema.db")))
	got := dbtest.New(t)
	assertSameSchema(t, want, schema(t, got))

	var n int
	if err := got.Get(&n, `SELECT count(*) FROM schema_migrations`); err != nil {
		t.Fatal(err)
	}
	if n != 77 {
		t.Fatalf("want 77 schema_migrations rows, got %d", n)
	}

	// The settings singleton is seeded by a migration, with a real route token.
	var token string
	if err := got.Get(&token, `SELECT route_token FROM settings`); err != nil {
		t.Fatal(err)
	}
	if len(token) != 36 {
		t.Fatalf("route_token should be a generated uuid, got %q", token)
	}
}

func TestMigrateIsNoopOnElixirDatabase(t *testing.T) {
	d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	before := schema(t, d)
	ran, err := d.Migrate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != 0 {
		t.Fatalf("expected no migrations on an up-to-date Elixir DB, ran %v", ran)
	}
	assertSameSchema(t, before, schema(t, d))
}

// A user on an older Elixir release jumps straight to Go: the remaining
// migrations must bring them to the same schema.
func TestMigrateFromPartiallyMigratedDatabase(t *testing.T) {
	migs, err := db.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, stop := range []int{1, 20, 45, 60, len(migs) - 1} {
		d := dbtestEmpty(t)
		if _, err := d.MigrateTo(context.Background(), migs[stop-1].Version); err != nil {
			t.Fatalf("stop %d: %v", stop, err)
		}
		// Insert a row mid-history, as a real user would have.
		if stop >= 3 {
			mustExec(t, d, `INSERT INTO media_profiles (name, output_path_template, inserted_at, updated_at) VALUES ('p', '{{title}}.{{ext}}', '2024-01-01T00:00:00Z', '2024-01-01T00:00:00Z')`)
		}
		ran, err := d.Migrate(context.Background())
		if err != nil {
			t.Fatalf("stop %d: %v", stop, err)
		}
		if len(ran) != len(migs)-stop {
			t.Fatalf("stop %d: ran %d migrations, want %d", stop, len(ran), len(migs)-stop)
		}
		assertSameSchema(t, schema(t, dbtest.CopyOf(t, dbtest.ElixirFixture("schema.db"))), schema(t, d))
	}
}

func TestForeignKeysAreEnforced(t *testing.T) {
	d := dbtest.New(t)
	var on int
	if err := d.Get(&on, `PRAGMA foreign_keys`); err != nil {
		t.Fatal(err)
	}
	if on != 1 {
		t.Fatal("foreign_keys pragma must be on")
	}
	_, err := d.Exec(`INSERT INTO tasks (job_id, inserted_at, updated_at) VALUES (999, 'x', 'x')`)
	if err == nil {
		t.Fatal("expected FK violation")
	}
}

func TestRegexpLike(t *testing.T) {
	d := dbtest.New(t)
	cases := []struct {
		src, pattern string
		want         int
	}{
		{"My Trailer", `(?i)^(?!.*trailer).*$`, 0}, // lookahead: RE2 can't do this
		{"Episode 4", `(?i)^(?!.*trailer).*$`, 1},
		{"abcabc", `(abc)\1`, 1}, // backreference
		{"Hello", `ell`, 1},      // unanchored, like PCRE
		{"Hello", `^ell`, 0},
	}
	for _, c := range cases {
		var got int
		if err := d.Get(&got, `SELECT regexp_like(?, ?)`, c.src, c.pattern); err != nil {
			t.Fatalf("%q %q: %v", c.src, c.pattern, err)
		}
		if got != c.want {
			t.Errorf("regexp_like(%q, %q) = %d, want %d", c.src, c.pattern, got, c.want)
		}
	}
	var res *int
	if err := d.Get(&res, `SELECT regexp_like(NULL, 'a')`); err != nil || res != nil {
		t.Errorf("NULL source should give NULL, got %v %v", res, err)
	}
	// Source validation relies on invalid patterns raising an error.
	if _, err := d.Exec(`SELECT regexp_like('', '(')`); err == nil {
		t.Error("invalid pattern should error")
	}
}

// Every timestamp and JSON value Ecto stored must survive a read/write
// round trip through our types byte for byte.
func TestEctoValueRoundTrip(t *testing.T) {
	f, err := os.Open(dbtest.ElixirFixture("golden/populated_values.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	timeCols := map[string]bool{"inserted_at": true, "updated_at": true, "uploaded_at": true,
		"media_downloaded_at": true, "media_redownloaded_at": true, "culled_at": true,
		"last_indexed_at": true, "marked_for_deletion_at": true, "details_updated_at": true}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	checked := 0
	for sc.Scan() {
		parts := strings.SplitN(sc.Text(), "|", 5)
		if len(parts) != 5 || parts[3] != "text" {
			continue
		}
		table, col, raw := parts[0], parts[2], unquote(parts[4])
		switch {
		case table == "oban_jobs":
			continue // covered by obanlite tests
		case table == "schema_migrations":
			roundTrip(t, table+"."+col, raw, &db.NaiveDateTime{})
		case table == "media_items" && col == "uploaded_at" && !strings.HasSuffix(raw, "Z"):
			// Legacy values are read correctly but rewritten in Ecto's current format.
			if _, err := db.ParseTime(raw); err != nil {
				t.Errorf("legacy %s.%s %q: %v", table, col, raw, err)
			}
			continue
		case timeCols[col]:
			roundTrip(t, table+"."+col, raw, &db.UTCDateTime{})
		case col == "download_cutoff_date":
			roundTrip(t, table+"."+col, raw, &db.Date{})
		case col == "subtitle_filepaths":
			roundTrip(t, table+"."+col, raw, &db.NestedStringArray{})
		case col == "sponsorblock_categories":
			roundTrip(t, table+"."+col, raw, &db.JSON[[]string]{})
		default:
			continue
		}
		checked++
	}
	if checked < 50 {
		t.Fatalf("only checked %d values; fixture parsing is broken", checked)
	}
}

func roundTrip(t *testing.T, label, raw string, v interface {
	Scan(any) error
}) {
	t.Helper()
	if err := v.Scan(raw); err != nil {
		t.Errorf("%s: scan %q: %v", label, raw, err)
		return
	}
	var out any
	var err error
	switch x := v.(type) {
	case *db.UTCDateTime:
		out, err = x.Value()
	case *db.Date:
		out, err = x.Value()
	case *db.NaiveDateTime:
		out, err = x.Value()
	case *db.NestedStringArray:
		out, err = x.Value()
	case *db.JSON[[]string]:
		out, err = x.Value()
	}
	if err != nil {
		t.Errorf("%s: value: %v", label, err)
		return
	}
	if out != raw {
		t.Errorf("%s: round trip %q -> %q", label, raw, out)
	}
}

// unquote reverses SQLite's quote() for TEXT values.
func unquote(s string) string {
	s = strings.TrimPrefix(s, "'")
	s = strings.TrimSuffix(s, "'")
	return strings.ReplaceAll(s, "''", "'")
}

func mustExec(t *testing.T, d *db.DB, q string, args ...any) {
	t.Helper()
	if _, err := d.Exec(q, args...); err != nil {
		t.Fatal(err)
	}
}

func dbtestEmpty(t *testing.T) *db.DB {
	t.Helper()
	d, err := db.Open(t.TempDir()+"/empty.db", db.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}
