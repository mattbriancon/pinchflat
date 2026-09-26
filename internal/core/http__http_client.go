package core

import "context"

// HTTPClientImpl is Pinchflat.HTTP.HTTPClient, the real HTTP client. It
// implements HTTPClient (app.go).
type HTTPClientImpl struct{}

var _ HTTPClient = (*HTTPClientImpl)(nil)

// get/3 — {:ok, body} | {:error, message}
func (c *HTTPClientImpl) Get(ctx context.Context, url string, headers KW, opts KW) (string, error) {
	panic("unported: Pinchflat.HTTP.HTTPClient.get/3")
}
