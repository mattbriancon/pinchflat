package web

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/app"
	"github.com/mattbriancon/pinchflat/internal/store"
)

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
	xml := app.OpmlFeedBuilderBuild(urlBase, sources)

	w.Header().Set("Content-Type", "application/opml+xml; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml))
}

// PodcastControllerRssFeed: rss_feed(conn, %{"uuid" => uuid})
func (s *Server) PodcastControllerRssFeed(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid := r.PathValue("uuid")

	// Fetch source by UUID: Repo.get_by!(Source, uuid: uuid)
	q := store.SourcesQueryNew().Where(sq.Eq{"s.uuid": uuid})
	source, err := store.One[store.Source](ctx, s.App.Q(ctx), q)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	urlBase := PageOf(ctx).BaseURL

	// Build RSS XML
	xml, err := s.App.RssFeedBuilderBuild(ctx, source, 2000, urlBase)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(xml))
}

// PodcastControllerFeedImage: feed_image(conn, %{"uuid" => uuid})
func (s *Server) PodcastControllerFeedImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid := r.PathValue("uuid")

	// Fetch source by UUID: Repo.get_by!(Source, uuid: uuid)
	q := store.SourcesQueryNew().Where(sq.Eq{"s.uuid": uuid})
	source, err := store.One[store.Source](ctx, s.App.Q(ctx), q)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Fetch media items for the source (used to find a fallback cover image):
	// MediaQuery.new() |> where(^dynamic(^MediaQuery.for_source(source) and ^MediaQuery.downloaded())) |> Repo.maybe_limit(1) |> Repo.all()
	mediaQuery := store.MediaQueryNew().
		Where(store.MediaQueryForSource(source.ID)).
		Where(store.MediaQueryDownloaded()).
		Map(func(b sq.SelectBuilder) sq.SelectBuilder {
			return b.Limit(1)
		})

	mediaItems, err := store.All[store.MediaItem](ctx, s.App.Q(ctx), mediaQuery)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Select cover image
	coverPath, err := s.App.PodcastHelpersSelectCoverImage(ctx, source, mediaItems)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Image not found"))
		return
	}

	sendFile(w, r, coverPath)
}

// PodcastControllerEpisodeImage: episode_image(conn, %{"uuid" => uuid})
func (s *Server) PodcastControllerEpisodeImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid := r.PathValue("uuid")

	// Fetch media item by UUID: Repo.get_by!(MediaItem, uuid: uuid)
	q := store.MediaQueryNew().Where(sq.Eq{"mi.uuid": uuid})
	mediaItem, err := store.One[store.MediaItem](ctx, s.App.Q(ctx), q)
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

	sendFile(w, r, *mediaItem.ThumbnailFilepath)
}

// sendFile is `put_resp_content_type(MIME.from_path(path)) |> send_file(200, path)`,
// answering 404 "Image not found" when the file is missing.
func sendFile(w http.ResponseWriter, r *http.Request, path string) {
	file, err := os.Open(path)
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	if t := mime.TypeByExtension(filepath.Ext(path)); t != "" {
		w.Header().Set("Content-Type", t)
	}
	http.ServeContent(w, r, path, stat.ModTime(), file)
}
