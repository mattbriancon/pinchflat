package web

import (
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
)

// MediaItemControllerShow shows a media item with details and associated tasks.
func (s *Server) MediaItemControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(URLParam(r, "id"), 10, 64)
	if err != nil {
		s.Fail(w, r, core.ErrNotFound)
		return
	}

	mediaItem, err := s.App.MediaGetMediaItem(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Preload source and tasks with jobs
	if mediaItem.Source == nil {
		source, err := s.App.SourcesGetSource(ctx, mediaItem.SourceID)
		if err != nil {
			s.Fail(w, r, core.ErrNotFound)
			return
		}
		mediaItem.Source = source
	}

	// Load tasks with jobs
	taskQuery := core.SQ.Select("*").From("tasks").Where(sq.Eq{"media_item_id": id}).OrderBy("id DESC")
	tasks, err := core.All[core.Task](ctx, s.App.Q(ctx), taskQuery)
	if err == nil {
		mediaItem.Tasks = tasks
		// Load jobs for each task
		for _, task := range tasks {
			if task != nil && task.Job == nil {
				job, _ := core.One[obanlite.Job](ctx, s.App.Q(ctx), core.SQ.Select("*").From("oban_jobs").Where(sq.Eq{"id": task.JobID}))
				task.Job = job
			}
		}
	}

	s.Render(w, r, http.StatusOK, LayoutApp, MediaItemsMediaItemHTMLShow(mediaItem))
}

// MediaItemControllerEdit renders the edit form for a media item.
func (s *Server) MediaItemControllerEdit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(URLParam(r, "id"), 10, 64)
	if err != nil {
		s.Fail(w, r, core.ErrNotFound)
		return
	}

	mediaItem, err := s.App.MediaGetMediaItem(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	cs := s.App.MediaChangeMediaItem(ctx, mediaItem, core.Attrs{})
	s.Render(w, r, http.StatusOK, LayoutApp, MediaItemsMediaItemHTMLEdit(mediaItem, cs))
}

// MediaItemControllerUpdate updates a media item.
func (s *Server) MediaItemControllerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(URLParam(r, "id"), 10, 64)
	if err != nil {
		s.Fail(w, r, core.ErrNotFound)
		return
	}

	mediaItem, err := s.App.MediaGetMediaItem(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	params := ParseForm(r, "media_item")
	updated, err := s.App.MediaUpdateMediaItem(ctx, mediaItem, params)
	if err != nil {
		// Handle changeset errors
		if csErr, ok := core.AsChangesetError(err); ok {
			csErr.Action = "update"
			s.Render(w, r, http.StatusOK, LayoutApp, MediaItemsMediaItemHTMLEdit(mediaItem, csErr))
			return
		}
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "Media Item updated successfully.")
	s.Redirect(w, r, P(ctx, "/sources/%v/media/%v", mediaItem.SourceID, updated.ID))
}

// MediaItemControllerDelete deletes the files associated with a media item.
func (s *Server) MediaItemControllerDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, err := strconv.ParseInt(URLParam(r, "id"), 10, 64)
	if err != nil {
		s.Fail(w, r, core.ErrNotFound)
		return
	}

	mediaItem, err := s.App.MediaGetMediaItem(ctx, id)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	prevent := r.URL.Query().Get("prevent_download") == "true"
	addlAttrs := core.Attrs{}
	if prevent {
		addlAttrs["prevent_download"] = true
	}

	_, err = s.App.MediaDeleteMediaFiles(ctx, mediaItem, addlAttrs)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	s.PutFlash(w, r, "info", "Files deleted successfully.")
	s.Redirect(w, r, P(ctx, "/sources/%v", mediaItem.SourceID))
}

// MediaItemControllerForceDownload enqueues a download task for a media item.
func (s *Server) MediaItemControllerForceDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mediaItemID, err := strconv.ParseInt(URLParam(r, "media_item_id"), 10, 64)
	if err != nil {
		s.Fail(w, r, core.ErrNotFound)
		return
	}

	mediaItem, err := s.App.MediaGetMediaItem(ctx, mediaItemID)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	_, err = s.App.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, core.Attrs{"force": true}, core.KW{})
	if err != nil {
		// Allow duplicate job errors to pass through silently
		if !strings.Contains(err.Error(), "duplicate") {
			slog.Warn("failed to kickoff download task", "err", err)
		}
	}

	s.PutFlash(w, r, "info", "Download task enqueued.")
	s.Redirect(w, r, P(ctx, "/sources/%v/media/%v", mediaItem.SourceID, mediaItem.ID))
}

// MediaItemControllerStream serves a media file with range request support.
// See: https://www.zeng.dev/post/2023-http-range-and-play-mp4-in-browser/
// Uses UUID instead of ID to avoid enumeration attacks since streaming is a public endpoint.
func (s *Server) MediaItemControllerStream(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	uuid := URLParam(r, "uuid")

	mediaItem, err := core.One[core.MediaItem](ctx, s.App.Q(ctx), core.SQ.Select("*").From("media_items").Where(sq.Eq{"uuid": uuid}))
	if err != nil || mediaItem == nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	filePath := ""
	if mediaItem.MediaFilepath != nil {
		filePath = *mediaItem.MediaFilepath
	}

	if filePath == "" || !fileExists(filePath) {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	fileSize := stat.Size()

	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	title := "media"
	if mediaItem.Title != nil {
		title = *mediaItem.Title
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Disposition", "inline; filename=\""+title+"\"")

	rangeStart, rangeEnd, valid := parseRange(r, fileSize)
	if !valid {
		// Invalid range, serve full file
		uuidStr := ""
		if mediaItem.UUID != nil {
			uuidStr = *mediaItem.UUID
		}
		slog.Debug("Invalid range request for media item", "uuid", uuidStr)
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		w.Header().Set("Content-Range", "bytes 0-"+strconv.FormatInt(fileSize-1, 10)+"/"+strconv.FormatInt(fileSize, 10))
		http.ServeContent(w, r, title, stat.ModTime(), file)
	} else {
		// Valid range, serve partial file
		uuidStr := ""
		if mediaItem.UUID != nil {
			uuidStr = *mediaItem.UUID
		}
		slog.Debug("Streaming media item", "uuid", uuidStr, "from", rangeStart, "to", rangeEnd)
		length := rangeEnd - rangeStart + 1
		w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
		w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(rangeStart, 10)+"-"+strconv.FormatInt(rangeEnd, 10)+"/"+strconv.FormatInt(fileSize, 10))
		w.WriteHeader(http.StatusPartialContent)
		file.Seek(rangeStart, 0)
		http.ServeContent(w, r, title, stat.ModTime(), file)
	}
}

// parseRange parses the Range header and returns the start and end positions.
// Returns (start, end, valid).
func parseRange(r *http.Request, fileSize int64) (int64, int64, bool) {
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		return 0, fileSize - 1, false
	}

	// Parse "bytes=0-100" format
	parts := strings.Split(rangeHeader, "=")
	if len(parts) != 2 || parts[0] != "bytes" {
		return 0, fileSize - 1, false
	}

	rangeParts := strings.Split(parts[1], "-")
	if len(rangeParts) != 2 {
		return 0, fileSize - 1, false
	}

	startStr, endStr := rangeParts[0], rangeParts[1]

	// Parse start position
	var start int64
	if startStr == "" {
		return 0, fileSize - 1, false
	}
	var err error
	start, err = strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return 0, fileSize - 1, false
	}

	// Parse end position
	var end int64
	if endStr == "" {
		// No end specified, serve to end of file
		end = fileSize - 1
	} else {
		end, err = strconv.ParseInt(endStr, 10, 64)
		if err != nil {
			return 0, fileSize - 1, false
		}

		// RFC7233: if end >= fileSize, serve to end of file
		if end >= fileSize {
			end = fileSize - 1
		}
	}

	if start < 0 || end < start {
		return 0, fileSize - 1, false
	}

	return start, end, true
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
