package store_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// The JSON given to user scripts must match what the Elixir encoders wrote.
func TestSourceAndProfileJSONMatchElixirGolden(t *testing.T) {
	d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	s := &store.Store{DB: d}
	ctx := context.Background()

	decode := func(b []byte) map[string]any {
		var m map[string]any
		must(t, json.Unmarshal(b, &m))
		return m
	}
	golden := func(name string) map[string]any {
		b, err := os.ReadFile(dbtest.ElixirFixture("golden/" + name))
		must(t, err)
		return decode(b)
	}

	t.Run("source", func(t *testing.T) {
		src, err := store.MustOne[store.Source](ctx, s.DB, store.From[store.Source]().Where(sq.Eq{"id": 1}))
		must(t, err)
		mustOK(t)(s.PreloadSourceMediaProfile(ctx, src))
		b, err := json.Marshal(src)
		must(t, err)
		if got, want := decode(b), golden("user_script_source.json"); !reflect.DeepEqual(got, want) {
			t.Errorf("source JSON differs\n got: %v\nwant: %v", got, want)
		}
	})

	t.Run("media profile", func(t *testing.T) {
		p, err := store.MustOne[store.MediaProfile](ctx, s.DB, store.From[store.MediaProfile]().Where(sq.Eq{"id": 2}))
		must(t, err)
		b, err := json.Marshal(p)
		must(t, err)
		if got, want := decode(b), golden("user_script_media_profile.json"); !reflect.DeepEqual(got, want) {
			t.Errorf("profile JSON differs\n got: %v\nwant: %v", got, want)
		}
	})
}
