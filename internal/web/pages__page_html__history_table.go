package web

import (
	"context"
	"math"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

// Port of lib/pinchflat_web/controllers/pages/page_html/history_table_live.ex.

const historyTableLimit = 5

// historyTableFetch fetches one media_state's page of history for the home
// page (mount/3 + handle_params in the old LiveView).
func historyTableFetch(ctx context.Context, app *core.App, mediaState string, page int) (records []*core.MediaItem, clampedPage, totalPages, totalRecordCount int, err error) {
	baseQuery := generateHistoryTableBaseQuery(mediaState)
	totalRecordCount, err = core.Scalar[int](ctx, app.Q(ctx),
		core.SQ.Select("COUNT(*)").FromSelect(baseQuery.B, "mi"))
	if err != nil {
		return nil, 0, 0, 0, err
	}
	totalPages = int(math.Max(math.Ceil(float64(totalRecordCount)/float64(historyTableLimit)), 1))
	clampedPage = fsutil.Clamp(page, 1, totalPages)

	offset := (clampedPage - 1) * historyTableLimit
	records, err = core.All[core.MediaItem](ctx, app.Q(ctx),
		baseQuery.B.Limit(uint64(historyTableLimit)).Offset(uint64(offset)))
	if err != nil {
		return nil, 0, 0, 0, err
	}

	// Preload source for each record
	for _, record := range records {
		source, _ := app.SourcesGetSource(ctx, record.SourceID)
		record.Source = source
	}

	return records, clampedPage, totalPages, totalRecordCount, nil
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
