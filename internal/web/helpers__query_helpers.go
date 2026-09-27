package web

// Small helpers for building "reload the page with new query params" links
// and forms, now that paging/sorting/search/tabs are plain navigations
// instead of htmx fragment requests (see STRATEGY.md decision 4).

import (
	"context"
	"net/url"
)

// currentQuery returns a copy of the current request's query params.
func currentQuery(ctx context.Context) url.Values {
	out := url.Values{}
	if req := PageOf(ctx).Request; req != nil {
		for k, v := range req.URL.Query() {
			cp := make([]string, len(v))
			copy(cp, v)
			out[k] = cp
		}
	}
	return out
}

// withQuery builds path with the current request's query params, overridden
// by overrides (a "" value deletes the param). It's used for links that
// change one thing (a page number, a sort column, the active tab) while
// keeping every other bit of page state (another table's page, a search
// term, ...) intact.
func withQuery(ctx context.Context, path string, overrides map[string]string) string {
	q := currentQuery(ctx)
	for k, v := range overrides {
		if v == "" {
			q.Del(k)
		} else {
			q.Set(k, v)
		}
	}
	if len(q) == 0 {
		return path
	}
	return path + "?" + q.Encode()
}

// preservedParams is the current query params minus except, for hidden
// inputs on a GET form so submitting it doesn't drop unrelated state (e.g.
// another tab's page, or which tab is selected).
func preservedParams(ctx context.Context, except ...string) url.Values {
	q := currentQuery(ctx)
	for _, e := range except {
		q.Del(e)
	}
	return q
}
