package web

// Port of lib/pinchflat_web/controllers/error_html.ex. The two embedded
// templates (404/500) are error_html__404.templ / error_html__500.templ,
// wired into render.go's NotFound/Fail. render/2's catch-all - "the default
// is to render a plain text page based on the template name" - is
// ErrorHTMLStatusMessage, kept for parity even though the Go port currently
// only ever renders 404 or 500 (render.go has no other caller).

import "net/http"

// ErrorHTMLStatusMessage is error_html.ex's render/2:
// Phoenix.Controller.status_message_from_template("404") => "Not Found".
func ErrorHTMLStatusMessage(template string) string {
	status := 0
	for _, c := range template {
		if c < '0' || c > '9' {
			status = 0
			break
		}
		status = status*10 + int(c-'0')
	}
	if status == 0 {
		return "Internal Server Error"
	}
	if msg := http.StatusText(status); msg != "" {
		return msg
	}
	return "Internal Server Error"
}
