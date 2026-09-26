package web

import (
	"mime"
	"net/http"
	"os"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
)

// Port of lib/pinchflat_web/controllers/podcasts/podcast_controller.ex.

// PodcastControllerOpmlFeed: opml_feed(conn, _params)
func (s *Server) PodcastControllerOpmlFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlBase := PageOf(ctx).BaseURL

	// Fetch sources without deletion marks
	sources, err := s.App.PodcastHelpersOpmlSources(ctx)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Build OPML XML
	xml := core.OpmlFeedBuilderBuild(urlBase, sources)

	w.Header().Set("Content-Type", "application/opml+xml")
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml))
}

// PodcastControllerRssFeed: rss_feed(conn, %{"uuid" => uuid})
func (s *Server) PodcastControllerRssFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid := URLParam(r, "uuid")

	// Fetch source by UUID: Repo.get_by!(Source, uuid: uuid)
	q := core.SourcesQueryNew().Where(sq.Eq{"s.uuid": uuid})
	source, err := core.One[core.Source](ctx, s.App.Q(ctx), q)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	urlBase := PageOf(ctx).BaseURL

	// Build RSS XML
	xml, err := s.App.RssFeedBuilderBuild(ctx, source, core.KW{
		core.Opt("limit", 2000),
		core.Opt("url_base", urlBase),
	})
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml")
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml))
}

// PodcastControllerFeedImage: feed_image(conn, %{"uuid" => uuid})
func (s *Server) PodcastControllerFeedImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid := URLParam(r, "uuid")

	// Fetch source by UUID: Repo.get_by!(Source, uuid: uuid)
	q := core.SourcesQueryNew().Where(sq.Eq{"s.uuid": uuid})
	source, err := core.One[core.Source](ctx, s.App.Q(ctx), q)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Fetch media items for the source (used to find a fallback cover image):
	// MediaQuery.new() |> where(^dynamic(^MediaQuery.for_source(source) and ^MediaQuery.downloaded())) |> Repo.maybe_limit(1) |> Repo.all()
	mediaQuery := core.MediaQueryNew().
		Where(core.MediaQueryForSource(source.ID)).
		Where(core.MediaQueryDownloaded()).
		Map(func(b sq.SelectBuilder) sq.SelectBuilder {
			return b.Limit(1)
		})

	mediaItems, err := core.All[core.MediaItem](ctx, s.App.Q(ctx), mediaQuery)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Select cover image
	filepath, err := s.App.PodcastHelpersSelectCoverImage(ctx, source, mediaItems)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Image not found"))
		return
	}

	// Serve the file: send_file(200, filepath)
	file, err := os.Open(filepath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Image not found"))
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Image not found"))
		return
	}

	mimeType := mime.TypeByExtension(filepath)
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}
	http.ServeContent(w, r, filepath, stat.ModTime(), file)
}

// PodcastControllerEpisodeImage: episode_image(conn, %{"uuid" => uuid})
func (s *Server) PodcastControllerEpisodeImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid := URLParam(r, "uuid")

	// Fetch media item by UUID: Repo.get_by!(MediaItem, uuid: uuid)
	q := core.MediaQueryNew().Where(sq.Eq{"mi.uuid": uuid})
	mediaItem, err := core.One[core.MediaItem](ctx, s.App.Q(ctx), q)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Check if thumbnail exists and is a valid file
	if mediaItem.ThumbnailFilepath == nil || *mediaItem.ThumbnailFilepath == "" {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Image not found"))
		return
	}

	filepath := *mediaItem.ThumbnailFilepath

	file, err := os.Open(filepath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Image not found"))
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Image not found"))
		return
	}

	mimeType := mime.TypeByExtension(filepath)
	if mimeType != "" {
		w.Header().Set("Content-Type", mimeType)
	}
	http.ServeContent(w, r, filepath, stat.ModTime(), file)
}
