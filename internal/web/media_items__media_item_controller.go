package web

import (
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/obanlite"
	"github.com/mattbriancon/pinchflat/internal/store"
)

// MediaItemControllerShow shows a media item with details and associated tasks.
func (s *Server) MediaItemControllerShow(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mediaItem, ok := loadOrFail(s, w, r, "id", s.App.GetMediaItem)
	if !ok {
		return
	}

	// Preload source and tasks with jobs
	if mediaItem.Source == nil {
		source, err := s.App.GetSource(ctx, mediaItem.SourceID)
		if err != nil {
			s.Fail(w, r, store.ErrNotFound)
			return
		}
		mediaItem.Source = source
	}

	// Load tasks with jobs
	taskQuery := store.SQ.Select("*").From("tasks").Where(sq.Eq{"media_item_id": mediaItem.ID}).OrderBy("id DESC")
	tasks, err := store.All[store.Task](ctx, s.App.Q(ctx), taskQuery)
	if err == nil {
		mediaItem.Tasks = tasks
		// Load jobs for each task
		for _, task := range tasks {
			if task != nil && task.Job == nil {
				job, _ := store.One[obanlite.Job](ctx, s.App.Q(ctx), store.SQ.Select("*").From("oban_jobs").Where(sq.Eq{"id": task.JobID}))
				task.Job = job
			}
		}
	}

	s.Render(w, r, http.StatusOK, LayoutApp, MediaItemsMediaItemHTMLShow(mediaItem))
}

// MediaItemControllerEdit renders the edit form for a media item.
func (s *Server) MediaItemControllerEdit(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mediaItem, ok := loadOrFail(s, w, r, "id", s.App.GetMediaItem)
	if !ok {
		return
	}

	cs := s.App.ChangeMediaItem(ctx, mediaItem, store.Attrs{})
	s.Render(w, r, http.StatusOK, LayoutApp, MediaItemsMediaItemHTMLEdit(mediaItem, cs))
}

// MediaItemControllerUpdate updates a media item.
func (s *Server) MediaItemControllerUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mediaItem, ok := loadOrFail(s, w, r, "id", s.App.GetMediaItem)
	if !ok {
		return
	}

	params := ParseForm(r, "media_item")
	updated, err := s.App.UpdateMediaItem(ctx, mediaItem, params)
	if err != nil {
		// Handle changeset errors
		if csErr, ok := store.AsChangesetError(err); ok {
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
	mediaItem, ok := loadOrFail(s, w, r, "id", s.App.GetMediaItem)
	if !ok {
		return
	}

	prevent := r.URL.Query().Get("prevent_download") == "true"
	addlAttrs := store.Attrs{}
	if prevent {
		addlAttrs["prevent_download"] = true
	}

	_, err := s.App.MediaDeleteMediaFiles(ctx, mediaItem, addlAttrs)
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
	mediaItem, ok := loadOrFail(s, w, r, "media_item_id", s.App.GetMediaItem)
	if !ok {
		return
	}

	_, err := s.App.MediaDownloadWorkerKickoffWithTask(ctx, mediaItem, store.Attrs{"force": true}, nil)
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

	mediaItem, err := store.One[store.MediaItem](ctx, s.App.Q(ctx), store.SQ.Select("*").From("media_items").Where(sq.Eq{"uuid": uuid}))
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

	title := ""
	if mediaItem.Title != nil {
		title = *mediaItem.Title
	}
	uuidStr := ""
	if mediaItem.UUID != nil {
		uuidStr = *mediaItem.UUID
	}

	// Plug.Conn.put_resp_content_type/2 always appends "; charset=utf-8"
	// unless a nil charset is passed explicitly.
	w.Header().Set("Content-Type", mimeType+"; charset=utf-8")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Disposition", "inline; filename=\""+title+"\"")

	rangeStart, rangeEnd, valid := parseRange(r, fileSize)
	if valid {
		slog.Debug("Streaming media item", "uuid", uuidStr, "from", rangeStart, "to", rangeEnd)
		length := rangeEnd - rangeStart + 1
		w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(rangeStart, 10)+"-"+strconv.FormatInt(rangeEnd, 10)+"/"+strconv.FormatInt(fileSize, 10))
		w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
		w.WriteHeader(http.StatusPartialContent)
		if _, err := file.Seek(rangeStart, io.SeekStart); err == nil {
			io.CopyN(w, file, length)
		}
	} else {
		slog.Debug("Invalid range request for media item", "uuid", uuidStr)
		w.Header().Set("Content-Range", "bytes 0-"+strconv.FormatInt(fileSize-1, 10)+"/"+strconv.FormatInt(fileSize, 10))
		w.Header().Set("Content-Length", strconv.FormatInt(fileSize, 10))
		w.WriteHeader(http.StatusOK)
		io.Copy(w, file)
	}
}

// parseRange is parse_range/2 + validate_range/3: it parses the Range
// header and returns the start and end positions. Returns (start, end,
// valid); valid is false for a missing or invalid header (RFC7233's
// "ignore, serve the full file" case, not a 416).
func parseRange(r *http.Request, fileSize int64) (int64, int64, bool) {
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		return 0, 0, false
	}

	// ["bytes", range] <- String.split(range_header, "=")
	parts := strings.Split(rangeHeader, "=")
	if len(parts) != 2 || parts[0] != "bytes" {
		return 0, 0, false
	}

	// [start_pos, end_pos] <- String.split(range, "-")
	rangeParts := strings.Split(parts[1], "-")
	if len(rangeParts) != 2 {
		return 0, 0, false
	}

	startStr, endStr := rangeParts[0], rangeParts[1]
	start, errStart := strconv.ParseInt(startStr, 10, 64)
	end, errEnd := strconv.ParseInt(endStr, 10, 64)

	switch {
	case errStart != nil:
		// {:error, :error} (or the unmatched {:error, {end_pos, _}} case)
		return 0, 0, false
	case errEnd != nil:
		// {{start_pos, _}, :error} -> {:ok, {start_pos, file_size - 1}}
		return start, fileSize - 1, true
	case end >= fileSize:
		// end_pos >= file_size -> {:ok, {start_pos, file_size - 1}}
		return start, fileSize - 1, true
	default:
		return start, end, true
	}
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
