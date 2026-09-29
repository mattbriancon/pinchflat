package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

// HTTPClientImpl is Pinchflat.HTTP.HTTPClient, the real HTTP client. It
// implements HTTPClient (app.go).
type HTTPClientImpl struct{}

var _ HTTPClient = (*HTTPClientImpl)(nil)

// get/3 — {:ok, body} | {:error, message}
func (c *HTTPClientImpl) Get(ctx context.Context, url string, headers http.Header) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %v", err)
	}

	req.Header = headers.Clone()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %v", err)
	}

	if resp.StatusCode == 200 {
		return string(body), nil
	}

	return "", fmt.Errorf("HTTP request failed with status code %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
}
