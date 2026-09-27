package web

import (
	"math"
	"net/http"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
)

// Port of lib/pinchflat_web/controllers/pages/page_html/history_table_live.ex.

const historyTableLiveLimit = 5

// HistoryTableLiveRender renders the history table as an htmx fragment.
func (s *Server) HistoryTableLiveRender(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	mediaState := r.URL.Query().Get("media_state")
	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil {
			page = p
		}
	}

	baseQuery := generateHistoryTableBaseQuery(mediaState)
	totalRecordCount, err := core.Scalar[int](ctx, s.App.Q(ctx),
		core.SQ.Select("COUNT(*)").FromSelect(baseQuery.B, "mi"))
	if err != nil {
		s.Fail(w, r, err)
		return
	}
	totalPages := int(math.Max(math.Ceil(float64(totalRecordCount)/float64(historyTableLiveLimit)), 1))
	clampedPage := core.NumberUtilsClamp(page, 1, totalPages)

	offset := (clampedPage - 1) * historyTableLiveLimit
	records, err := core.All[core.MediaItem](ctx, s.App.Q(ctx),
		baseQuery.B.Limit(uint64(historyTableLiveLimit)).Offset(uint64(offset)))
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	// Preload source for each record
	for _, record := range records {
		source, _ := s.App.SourcesGetSource(ctx, record.SourceID)
		record.Source = source
	}

	s.RenderFragment(w, r, http.StatusOK, PagesPageHTMLHistoryTableLiveContent(ctx, mediaState, clampedPage, totalPages, records, totalRecordCount))
}

// generateHistoryTableBaseQuery creates the base query for the history table.
func generateHistoryTableBaseQuery(mediaState string) *core.MediaQ {
	q := core.MediaQueryNew().RequireAssoc("media_profile")
	if mediaState == "pending" {
		q = q.Where(core.MediaQueryPending())
	} else {
		q = q.Where(core.MediaQueryDownloaded())
	}
	return q.Map(func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.OrderBy("mi.id DESC")
	})
}
