package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// statementSeparator splits statements inside a migration file. Plain ';'
// can't be used because trigger bodies contain semicolons.
const statementSeparator = "-- +statement"

// Migration is one embedded migration file.
type Migration struct {
	Version int64
	Name    string
	SQL     []string
}

// Migrations returns every embedded migration in version order.
func Migrations() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return nil, err
	}
	var out []Migration
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".sql") || len(name) < 15 {
			continue
		}
		v, err := strconv.ParseInt(name[:14], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad migration filename %s: %w", name, err)
		}
		body, err := migrationFS.ReadFile("migrations/" + name)
		if err != nil {
			return nil, err
		}
		var stmts []string
		for _, s := range strings.Split(string(body), statementSeparator) {
			if s = strings.TrimSpace(s); s != "" {
				stmts = append(stmts, s)
			}
		}
		out = append(out, Migration{Version: v, Name: strings.TrimSuffix(name[15:], ".sql"), SQL: stmts})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Version < out[j].Version })
	return out, nil
}

// Migrate applies every embedded migration that isn't recorded in Ecto's
// schema_migrations table, each in its own transaction, recording versions
// the way Ecto does. It is a no-op on a database the Elixir app already
// migrated. It returns the versions it applied.
func (d *DB) Migrate(ctx context.Context) ([]int64, error) {
	return d.MigrateTo(ctx, 0)
}

// MigrateTo is Migrate, stopping after version target (0 means all).
func (d *DB) MigrateTo(ctx context.Context, target int64) ([]int64, error) {
	// Ecto's exact DDL, so a fresh Go database is identical to an Elixir one.
	if _, err := d.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS "schema_migrations" ("version" INTEGER PRIMARY KEY, "inserted_at" TEXT)`); err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}

	var appliedList []int64
	if err := d.SelectContext(ctx, &appliedList, `SELECT version FROM schema_migrations`); err != nil {
		return nil, err
	}
	applied := make(map[int64]bool, len(appliedList))
	for _, v := range appliedList {
		applied[v] = true
	}

	migs, err := Migrations()
	if err != nil {
		return nil, err
	}

	var ran []int64
	for _, m := range migs {
		if target != 0 && m.Version > target {
			break
		}
		if applied[m.Version] {
			continue
		}
		err := d.InTx(ctx, func(tx *Tx) error {
			for _, stmt := range m.SQL {
				if _, err := tx.ExecContext(ctx, stmt); err != nil {
					return fmt.Errorf("%s", firstLine(stmt)+": "+err.Error())
				}
			}
			_, err := tx.ExecContext(ctx, `INSERT INTO "schema_migrations" ("version","inserted_at") VALUES (?,?)`, m.Version, NaiveDateTime{Now().Time})
			return err
		})
		if err != nil {
			return ran, fmt.Errorf("migration %d_%s: %w", m.Version, m.Name, err)
		}
		slog.Info("applied migration", "version", m.Version, "name", m.Name)
		ran = append(ran, m.Version)
	}
	return ran, nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
