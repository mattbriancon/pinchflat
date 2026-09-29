package store

// Store holds the database and job queue every query/CRUD helper needs.
// Hand-written W0 infrastructure; not a manifest row.

import (
	"context"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// Store is the persistence layer: the Repo and Oban.
type Store struct {
	DB   *db.DB
	Oban *obanlite.Oban
}

// Q returns the querier to use: the transaction in ctx if one was started
// with InTx, else the database. Always write `s.Q(ctx)` where Elixir used Repo.
func (s *Store) Q(ctx context.Context) db.Querier {
	if tx, ok := ctx.Value(txKey{}).(*db.Tx); ok {
		return tx
	}
	return s.DB
}

type txKey struct{}

// InTx runs fn in a transaction (Repo.transaction/1). Calls to s.Q(ctx)
// inside fn use the transaction. Nested calls reuse the outer transaction.
func (s *Store) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*db.Tx); ok {
		return fn(ctx)
	}
	return s.DB.InTx(ctx, func(tx *db.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}
