package core_test

// Go-only compatibility test: RSS and OPML output for the Elixir-written
// populated.db must be byte-identical to what the Elixir app produced
// (testdata/elixir/golden/*.xml), apart from the channel's build time.

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

var channelDates = regexp.MustCompile(`<(lastBuildDate|pubDate)>[^<]*</(lastBuildDate|pubDate)>`)

// normalizeBuildTime blanks the first two date tags (channel lastBuildDate
// and pubDate, which are "now"); item pubDates are data and must match.
func normalizeBuildTime(s string) string {
	n := 0
	return channelDates.ReplaceAllStringFunc(s, func(m string) string {
		n++
		if n > 2 {
			return m
		}
		return channelDates.ReplaceAllString(m, "<$1>NOW</$2>")
	})
}

func goldenApp(t *testing.T) (*core.App, context.Context) {
	d := dbtest.CopyOf(t, dbtest.ElixirFixture("populated.db"))
	app := &core.App{Store: &store.Store{DB: d, Oban: obanlite.New(d)}, Config: core.Config{Env: "test"}}
	app.RegisterWorkers()
	return app, context.Background()
}

func TestPodcastsGoldenOutput(t *testing.T) {
	app, ctx := goldenApp(t)
	for _, c := range []struct {
		file string
		id   int64
	}{{"rss_channel.xml", 1}, {"rss_playlist.xml", 2}} {
		t.Run(c.file, func(t *testing.T) {
			want, err := os.ReadFile(dbtest.ElixirFixture("golden/" + c.file))
			if err != nil {
				t.Fatal(err)
			}
			source, err := store.MustOne[store.Source](ctx, app.DB, store.From[store.Source]().Where(sq.Eq{"id": c.id}))
			if err != nil {
				t.Fatal(err)
			}
			got, err := app.RssFeedBuilderBuild(ctx, source, store.KW{store.Opt("url_base", "http://pinchflat.test:8945")})
			if err != nil {
				t.Fatal(err)
			}
			if normalizeBuildTime(got) != normalizeBuildTime(string(want)) {
				t.Errorf("RSS differs from Elixir output\n%s", firstDiff(normalizeBuildTime(string(want)), normalizeBuildTime(got)))
			}
		})
	}
	t.Run("opml.xml", func(t *testing.T) {
		want, err := os.ReadFile(dbtest.ElixirFixture("golden/opml.xml"))
		if err != nil {
			t.Fatal(err)
		}
		sources, err := app.PodcastHelpersOpmlSources(ctx)
		if err != nil {
			t.Fatal(err)
		}
		got := core.OpmlFeedBuilderBuild("http://pinchflat.test:8945", sources)
		if got != string(want) {
			t.Errorf("OPML differs from Elixir output\n%s", firstDiff(string(want), got))
		}
	})
}

func firstDiff(want, got string) string {
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			return "line " + itoa(i+1) + ":\n want: " + strings.TrimRight(w, " ") + "\n  got: " + strings.TrimRight(g, " ")
		}
	}
	return "(identical lines; trailing whitespace or newline differs)"
}

func itoa(i int) string {
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if s == "" {
		return "0"
	}
	return s
}
