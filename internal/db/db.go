// Package db opens Pinchflat's SQLite database with the same settings the
// Elixir app (ecto_sqlite3) used, and provides the SQL functions the schema
// and queries rely on that used to come from the sqlean extension.
package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/dlclark/regexp2"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"modernc.org/sqlite"
)

// DB wraps sqlx.DB. All app code talks to the database through this type.
type DB struct {
	*sqlx.DB
}

// Querier is satisfied by both *DB and *Tx so helpers can run inside or
// outside a transaction.
type Querier interface {
	sqlx.ExtContext
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
}

// Tx is a transaction; it satisfies Querier.
type Tx struct {
	*sqlx.Tx
}

// Options configures Open.
type Options struct {
	// JournalMode mirrors the JOURNAL_MODE env var. Defaults to "wal".
	JournalMode string
	// MaxOpenConns defaults to 5, matching the Elixir Repo pool_size.
	MaxOpenConns int
}

var registerOnce sync.Once

// Open opens (creating if needed) the database at path. Use ":memory:" only
// for throwaway tests; most tests should use a temp file.
func Open(path string, opts Options) (*DB, error) {
	registerOnce.Do(registerFunctions)

	if opts.JournalMode == "" {
		opts.JournalMode = "wal"
	}
	if opts.MaxOpenConns == 0 {
		opts.MaxOpenConns = 5
	}

	// Pragmas match ecto_sqlite3 0.19 defaults. foreign_keys is essential:
	// tasks -> oban_jobs and the metadata tables rely on ON DELETE CASCADE.
	q := url.Values{}
	for _, p := range []string{
		"foreign_keys(1)",
		"busy_timeout(5000)",
		"journal_mode(" + strings.ToUpper(opts.JournalMode) + ")",
		"synchronous(NORMAL)",
		"temp_store(MEMORY)",
		"cache_size(-64000)",
	} {
		q.Add("_pragma", p)
	}
	// BEGIN IMMEDIATE avoids SQLITE_BUSY upgrades from read to write locks.
	q.Set("_txlock", "immediate")

	dsn := "file:" + path + "?" + q.Encode()
	sqldb, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	sqldb.SetMaxOpenConns(opts.MaxOpenConns)
	if err := sqldb.Ping(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("ping %s: %w", path, err)
	}
	return &DB{DB: sqldb}, nil
}

// InTx runs fn inside a transaction, committing on success and rolling back
// on error or panic.
func (d *DB) InTx(ctx context.Context, fn func(tx *Tx) error) (err error) {
	sqltx, err := d.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	tx := &Tx{Tx: sqltx}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
		if err != nil {
			tx.Rollback()
			return
		}
		err = tx.Commit()
	}()
	return fn(tx)
}

// registerFunctions installs Go replacements for the sqlean functions the app
// used: regexp_like (queries + Source validation) and gen_random_uuid (two
// historical data migrations).
func registerFunctions() {
	must(sqlite.RegisterDeterministicScalarFunction("regexp_like", 2, regexpLike))
	must(sqlite.RegisterScalarFunction("gen_random_uuid", 0, func(*sqlite.FunctionContext, []driver.Value) (driver.Value, error) {
		return uuid.NewString(), nil
	}))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

var regexCache sync.Map // pattern -> *regexp2.Regexp

// CompileRegex compiles a pattern with the same PCRE-like semantics sqlean's
// regexp_like used (lookarounds, backreferences, inline flags). Go's regexp
// package (RE2) would reject patterns users may already have saved.
func CompileRegex(pattern string) (*regexp2.Regexp, error) {
	if re, ok := regexCache.Load(pattern); ok {
		return re.(*regexp2.Regexp), nil
	}
	re, err := regexp2.Compile(pattern, regexp2.None)
	if err != nil {
		return nil, err
	}
	regexCache.Store(pattern, re)
	return re, nil
}

// regexpLike(source, pattern): 1 if pattern matches anywhere in source, 0 if
// not, NULL if either argument is NULL, error if the pattern is invalid.
func regexpLike(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
	if args[0] == nil || args[1] == nil {
		return nil, nil
	}
	source, pattern := toString(args[0]), toString(args[1])
	re, err := CompileRegex(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern: %w", err)
	}
	ok, err := re.MatchString(source)
	if err != nil {
		return nil, err
	}
	if ok {
		return int64(1), nil
	}
	return int64(0), nil
}

func toString(v driver.Value) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprint(t)
	}
}
