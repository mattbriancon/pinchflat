package app

import (
	"fmt"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// OpmlFeedBuilder builds OPML feeds for a list of sources.

// OpmlFeedBuilderBuild/2
func OpmlFeedBuilderBuild(urlBase string, sources []*store.Source) string {
	var sourcesXML []string
	for _, source := range sources {
		sourceRoute := opmlFeedBuilderSourceRoute(urlBase, source)
		// Each item is an Elixir heredoc, so it ends with a newline.
		outline := fmt.Sprintf("<outline type=\"rss\" text=\"%s\" xmlUrl=\"%s\" />\n", xmlSafe(source.CustomName), xmlSafe(sourceRoute))
		sourcesXML = append(sourcesXML, outline)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<opml version="2.0">
  <head>
    <title>All Sources</title>
  </head>
  <body>
    %s
  </body>
</opml>
`, strings.Join(sourcesXML, "\n"))
}

// opmlFeedBuilderSourceRoute builds the URL route for a source's RSS feed
func opmlFeedBuilderSourceRoute(urlBase string, source *store.Source) string {
	uuid := ""
	if source.UUID != nil {
		uuid = *source.UUID
	}
	// Route: /sources/{uuid}/feed.xml
	// Use simple string concat for URLs to avoid filepath.Join collapsing slashes
	if !strings.HasSuffix(urlBase, "/") {
		urlBase += "/"
	}
	return urlBase + fmt.Sprintf("sources/%s/feed.xml", uuid)
}
