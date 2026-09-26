package core

import "context"

// HTTPClientGet/3
func (a *App) HTTPClientGet(ctx context.Context, url string, headers KW, opts KW) (string, error) {
	panic("unported: Pinchflat.HTTP.HTTPClient.get/3")
}

// HTTPClientParseHeaders/1
func HTTPClientParseHeaders(headers KW) map[string]string {
	panic("unported: Pinchflat.HTTP.HTTPClient.parse_headers/1")
}
