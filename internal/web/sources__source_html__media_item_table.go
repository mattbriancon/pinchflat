package web

// Rendered inline for each media_state tab on the source show page.

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

const mediaItemTableLimit = 10

// mediaItemTableData is this render's state.
type mediaItemTableData struct {
	Source              *core.Source
	MediaState          string
	Page                int
	TotalPages          int
	Records             []*core.MediaItem
	SearchTerm          string
	TotalRecordCount    int
	FilteredRecordCount int
}

func mediaItemTableWrapperID(sourceID int64, mediaState string) string {
	return fmt.Sprintf("media-table-%d-%s", sourceID, mediaState)
}

// mediaItemTablePageParam/mediaItemTableSearchParam are this media_state's
// own query params, so paging/searching one tab doesn't disturb another's
// (STRATEGY.md decision 4: no htmx).
func mediaItemTablePageParam(mediaState string) string   { return mediaState + "_page" }
func mediaItemTableSearchParam(mediaState string) string { return mediaState + "_q" }

// mediaItemTableFetch is mount/3 + fetch_pagination_attributes/3, reading
// page/search from this media_state's query params on r.
func (s *Server) mediaItemTableFetch(ctx context.Context, r *http.Request, source *core.Source, mediaState string) (*mediaItemTableData, error) {
	page, _ := strconv.Atoi(r.URL.Query().Get(mediaItemTablePageParam(mediaState)))
	if page < 1 {
		page = 1
	}
	searchTerm := r.URL.Query().Get(mediaItemTableSearchParam(mediaState))

	base := mediaItemTableBaseQuery(source, mediaState)

	totalRecordCount, err := core.Scalar[int](ctx, s.App.Q(ctx), core.SQ.Select("COUNT(*)").FromSelect(base.B, "sq"))
	if err != nil {
		return nil, err
	}
	if totalRecordCount == 0 {
		return &mediaItemTableData{Source: source, MediaState: mediaState, TotalRecordCount: 0}, nil
	}

	trimmed := strings.TrimSpace(searchTerm)
	filtered := base
	if trimmed != "" {
		term := trimmed
		filtered = base.MatchingSearchTerm(&term)
	}

	filteredRecordCount, err := core.Scalar[int](ctx, s.App.Q(ctx), core.SQ.Select("COUNT(*)").FromSelect(filtered.B, "sq"))
	if err != nil {
		return nil, err
	}

	totalPages := fsutil.Clamp((filteredRecordCount+mediaItemTableLimit-1)/mediaItemTableLimit, 1, 1<<30)
	if filteredRecordCount == 0 {
		totalPages = 1
	}
	page = fsutil.Clamp(page, 1, totalPages)
	offset := (page - 1) * mediaItemTableLimit

	recordsQuery := filtered.Map(func(b sq.SelectBuilder) sq.SelectBuilder {
		if trimmed != "" {
			b = b.OrderBy("rank DESC")
		}
		return b.OrderBy("mi.uploaded_at DESC").Limit(mediaItemTableLimit).Offset(uint64(offset))
	})
	records, err := core.All[core.MediaItem](ctx, s.App.Q(ctx), recordsQuery.B)
	if err != nil {
		return nil, err
	}

	return &mediaItemTableData{
		Source:              source,
		MediaState:          mediaState,
		Page:                page,
		TotalPages:          totalPages,
		Records:             records,
		SearchTerm:          searchTerm,
		TotalRecordCount:    totalRecordCount,
		FilteredRecordCount: filteredRecordCount,
	}, nil
}

// generate_base_query/2.
func mediaItemTableBaseQuery(source *core.Source, mediaState string) *core.MediaQ {
	q := core.MediaQueryNew()
	switch mediaState {
	case "pending":
		return q.RequireAssoc("media_profile").Where(sq.And{core.MediaQueryForSource(source.ID), core.MediaQueryPending()})
	case "downloaded":
		return q.Where(sq.And{core.MediaQueryForSource(source.ID), core.MediaQueryDownloaded()})
	case "other":
		return q.RequireAssoc("media_profile").Where(sq.And{
			core.MediaQueryForSource(source.ID),
			core.Not(core.MediaQueryDownloaded()),
			core.Not(core.MediaQueryPending()),
		})
	default:
		return q.Where(core.MediaQueryForSource(source.ID))
	}
}

// mediaItemTablePageURL builds the prev (delta -1) / next (delta 1) page
// link (keeping this tab active and every other tab's state), "" when that
// direction isn't available.
func mediaItemTablePageURL(ctx context.Context, data *mediaItemTableData, delta int) string {
	if delta < 0 && data.Page <= 1 {
		return ""
	}
	if delta > 0 && data.Page >= data.TotalPages {
		return ""
	}
	return withQuery(ctx, P(ctx, "/sources/%v", data.Source.ID), map[string]string{
		mediaItemTablePageParam(data.MediaState): strconv.Itoa(data.Page + delta),
		"tab":                                    data.MediaState,
	})
}

// mediaItemTableRows/Columns build the generic TableTable props.
func mediaItemTableRows(records []*core.MediaItem) []any {
	out := make([]any, len(records))
	for i, r := range records {
		out[i] = r
	}
	return out
}

func mediaItemTableColumns(data *mediaItemTableData) []TableColumn {
	cols := []TableColumn{
		{
			Label: "Title",
			Class: "cell-wrap",
			Render: func(row any) templ.Component {
				return mediaItemTableTitleCell(data, row.(*core.MediaItem))
			},
		},
	}
	if data.MediaState == "other" {
		cols = append(cols, TableColumn{
			Label: "Manually Ignored?",
			Render: func(row any) templ.Component {
				return mediaItemTableIgnoredCell(row.(*core.MediaItem))
			},
		})
	}
	cols = append(cols,
		TableColumn{
			Label: "Upload Date",
			Render: func(row any) templ.Component {
				return mediaItemTableUploadDateCell(row.(*core.MediaItem))
			},
		},
		TableColumn{
			Label: "",
			Class: "flex justify-end",
			Render: func(row any) templ.Component {
				return mediaItemTableEditCell(data, row.(*core.MediaItem))
			},
		},
	)
	return cols
}
