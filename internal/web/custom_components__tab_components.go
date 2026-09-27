package web

// Port of lib/pinchflat_web/components/custom_components/tab_components.ex
// (the renderable half is custom_components__tab_components.templ).

import (
	"context"

	"github.com/a-h/templ"
)

// TabTab is the :tab slot on tabbed_layout/1.
type TabTab struct {
	ID      string
	Title   string
	Content templ.Component
}

// tabFirstID is hd(assigns.tab).id.
func tabFirstID(tabs []TabTab) string { return tabs[0].ID }

// tabInitialID is the tab selected on first render: the ?tab= query param,
// when it names one of this layout's tabs (so a reload from a paging/search
// link inside a tab lands back on it), else the first tab.
func tabInitialID(ctx context.Context, tabs []TabTab) string {
	want := Param(ctx, "tab")
	for _, t := range tabs {
		if t.ID == want {
			return want
		}
	}
	return tabFirstID(tabs)
}
