package web

// Port of lib/pinchflat_web/controllers/sources/source_html/media_item_table_live.ex.
// See router.go: GET /_live/sources/{id}/media?media_state=&page=&q=.

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/core"
)

const mediaItemTableLiveLimit = 10

// mediaItemTableLiveData is the LiveView's assigns.
type mediaItemTableLiveData struct {
	Source              *core.Source
	MediaState          string
	Page                int
	TotalPages          int
	Records             []*core.MediaItem
	SearchTerm          string
	TotalRecordCount    int
	FilteredRecordCount int
}

// MediaItemTableLiveRender is PinchflatWeb.Sources.MediaItemTableLive: the
// "Pending"/"Downloaded"/"Other" media table on the source show page.
func (s *Server) MediaItemTableLiveRender(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sourceID, _ := strconv.ParseInt(URLParam(r, "id"), 10, 64)

	source, err := s.App.SourcesGetSource(ctx, sourceID)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	mediaState := r.URL.Query().Get("media_state")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	searchTerm := r.URL.Query().Get("q")

	data, err := s.mediaItemTableLiveFetch(ctx, source, mediaState, page, searchTerm)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	s.RenderFragment(w, r, http.StatusOK, SourceHTMLMediaItemTableLive(data))
}

// mediaItemTableLiveFetch is mount/3 + fetch_pagination_attributes/3.
func (s *Server) mediaItemTableLiveFetch(ctx context.Context, source *core.Source, mediaState string, page int, searchTerm string) (*mediaItemTableLiveData, error) {
	base := mediaItemTableLiveBaseQuery(source, mediaState)

	totalRecordCount, err := core.Scalar[int](ctx, s.App.Q(ctx), core.SQ.Select("COUNT(*)").FromSelect(base.B, "sq"))
	if err != nil {
		return nil, err
	}
	if totalRecordCount == 0 {
		return &mediaItemTableLiveData{Source: source, MediaState: mediaState, TotalRecordCount: 0}, nil
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

	totalPages := core.NumberUtilsClamp((filteredRecordCount+mediaItemTableLiveLimit-1)/mediaItemTableLiveLimit, 1, 1<<30)
	if filteredRecordCount == 0 {
		totalPages = 1
	}
	page = core.NumberUtilsClamp(page, 1, totalPages)
	offset := (page - 1) * mediaItemTableLiveLimit

	recordsQuery := filtered.Map(func(b sq.SelectBuilder) sq.SelectBuilder {
		if trimmed != "" {
			b = b.OrderBy("rank DESC")
		}
		return b.OrderBy("mi.uploaded_at DESC").Limit(mediaItemTableLiveLimit).Offset(uint64(offset))
	})
	records, err := core.All[core.MediaItem](ctx, s.App.Q(ctx), recordsQuery.B)
	if err != nil {
		return nil, err
	}

	return &mediaItemTableLiveData{
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
func mediaItemTableLiveBaseQuery(source *core.Source, mediaState string) *core.MediaQ {
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

func mediaItemTableLiveFragmentURL(ctx context.Context, sourceID int64, mediaState string, page int, searchTerm string) string {
	u := P(ctx, "/_live/sources/%v/media", sourceID) + "?media_state=" + url.QueryEscape(mediaState) + "&page=" + strconv.Itoa(page)
	if searchTerm != "" {
		u += "&q=" + url.QueryEscape(searchTerm)
	}
	return u
}

func mediaItemTableLiveTargetAttrs(ctx context.Context, data *mediaItemTableLiveData, page int) templ.Attributes {
	return templ.Attributes{
		"hx-get":    mediaItemTableLiveFragmentURL(ctx, data.Source.ID, data.MediaState, page, data.SearchTerm),
		"hx-target": "#" + mediaItemTableLiveWrapperID(data.Source.ID, data.MediaState),
		"hx-swap":   "outerHTML",
	}
}

// mediaItemTableLiveReloadAttrs is the "reload_page" event: with no PubSub
// socket to broadcast to, it just re-fetches this one fragment (see
// CONVENTIONS "LiveViews -> htmx").
func mediaItemTableLiveReloadAttrs(ctx context.Context, data *mediaItemTableLiveData) templ.Attributes {
	page := data.Page
	if page == 0 {
		page = 1
	}
	return mediaItemTableLiveTargetAttrs(ctx, data, page)
}

// mediaItemTableLivePageAttrs is the "page_change" event for direction
// "inc"/"dec".
func mediaItemTableLivePageAttrs(ctx context.Context, data *mediaItemTableLiveData, direction string) templ.Attributes {
	newPage := core.NumberUtilsClamp(core.NumberUtilsClamp(data.Page, 1, data.TotalPages)+map[string]int{"inc": 1, "dec": -1}[direction], 1, data.TotalPages)
	return mediaItemTableLiveTargetAttrs(ctx, data, newPage)
}

// mediaItemTableLiveRows/Columns build the generic TableTable props.
func mediaItemTableLiveRows(records []*core.MediaItem) []any {
	out := make([]any, len(records))
	for i, r := range records {
		out[i] = r
	}
	return out
}

func mediaItemTableLiveColumns(data *mediaItemTableLiveData) []TableColumn {
	cols := []TableColumn{
		{
			Label: "Title",
			Class: "cell-wrap",
			Render: func(row any) templ.Component {
				return mediaItemTableLiveTitleCell(data, row.(*core.MediaItem))
			},
		},
	}
	if data.MediaState == "other" {
		cols = append(cols, TableColumn{
			Label: "Manually Ignored?",
			Render: func(row any) templ.Component {
				return mediaItemTableLiveIgnoredCell(row.(*core.MediaItem))
			},
		})
	}
	cols = append(cols,
		TableColumn{
			Label: "Upload Date",
			Render: func(row any) templ.Component {
				return mediaItemTableLiveUploadDateCell(row.(*core.MediaItem))
			},
		},
		TableColumn{
			Label: "",
			Class: "flex justify-end",
			Render: func(row any) templ.Component {
				return mediaItemTableLiveEditCell(data, row.(*core.MediaItem))
			},
		},
	)
	return cols
}
