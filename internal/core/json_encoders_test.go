package core_test

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
)

// The JSON given to user scripts must match what the Elixir encoders wrote.
func TestSourceAndProfileJSONMatchElixirGolden(t *testing.T) {
	app, ctx := goldenApp(t)
	decode := func(b []byte) map[string]any {
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	golden := func(name string) map[string]any {
		b, err := os.ReadFile(dbtest.ElixirFixture("golden/" + name))
		if err != nil {
			t.Fatal(err)
		}
		return decode(b)
	}

	t.Run("source", func(t *testing.T) {
		s, err := core.MustOne[core.Source](ctx, app.DB, core.From[core.Source]().Where(sq.Eq{"id": 1}))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := app.PreloadSourceMediaProfile(ctx, s); err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := decode(b), golden("user_script_source.json"); !reflect.DeepEqual(got, want) {
			t.Errorf("source JSON differs\n got: %v\nwant: %v", got, want)
		}
	})
	t.Run("media profile", func(t *testing.T) {
		p, err := core.MustOne[core.MediaProfile](ctx, app.DB, core.From[core.MediaProfile]().Where(sq.Eq{"id": 2}))
		if err != nil {
			t.Fatal(err)
		}
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if got, want := decode(b), golden("user_script_media_profile.json"); !reflect.DeepEqual(got, want) {
			t.Errorf("profile JSON differs\n got: %v\nwant: %v", got, want)
		}
	})
}
