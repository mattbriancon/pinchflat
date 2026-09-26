package web

// Port of lib/pinchflat_web/components/custom_components/tab_components.ex
// (the renderable half is custom_components__tab_components.templ).

import "github.com/a-h/templ"

// TabTab is the :tab slot on tabbed_layout/1.
type TabTab struct {
	ID      string
	Title   string
	Content templ.Component
}

// tabFirstID is hd(assigns.tab).id.
func tabFirstID(tabs []TabTab) string { return tabs[0].ID }
