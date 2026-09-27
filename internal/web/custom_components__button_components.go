package web

// Port of lib/pinchflat_web/components/custom_components/button_components.ex
// (the renderable half is custom_components__button_components.templ).

import "github.com/a-h/templ"

// ButtonButtonOption is the :option slot on button_dropdown/1.
type ButtonButtonOption struct {
	Content templ.Component
}
