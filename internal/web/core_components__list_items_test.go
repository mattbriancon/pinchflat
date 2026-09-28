package web_test

import (
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/app/apptest"
	"github.com/mattbriancon/pinchflat/internal/store"
	"github.com/mattbriancon/pinchflat/internal/web"
)

func TestCoreListItemsFromMap(t *testing.T) {
	t.Run("renders schema fields but not associations", func(t *testing.T) {
		ta := apptest.NewApp(t)
		source := apptest.SourceFixture(t, ta, store.Attrs{"original_url": "https://www.youtube.com/@x"})
		source, _ = ta.App.PreloadSourceMediaProfile(ta.Ctx, source)

		html, err := templ.ToGoHTML(context.Background(), web.CoreListItemsFromMap(source))
		if err != nil {
			t.Fatal(err)
		}
		s := string(html)
		for _, want := range []string{"<strong>custom_name:</strong>", "<strong>inserted_at:</strong>", `href="https://www.youtube.com/@x"`} {
			if !strings.Contains(s, want) {
				t.Errorf("missing %q", want)
			}
		}
		if strings.Contains(s, "<strong>media_profile:</strong>") {
			t.Error("the preloaded media_profile association should be filtered out")
		}
		if !strings.Contains(s, source.InsertedAt.UTC().Format("2006-01-02 15:04:05Z")) {
			t.Error("datetimes should render like DateTime.to_string/1")
		}
	})
}
