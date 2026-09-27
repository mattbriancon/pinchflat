package web_test

import (
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestSearchController_Show(t *testing.T) {
	t.Run("renders the page", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/search")

		if !strings.Contains(res.HTML(t, 200), "Results") {
			t.Error("expected 'Results' in HTML")
		}
	})
}
