package web

// Port of lib/pinchflat_web/controllers/sources/source_live/index_table_live.ex.
// See router.go: GET /_live/sources?sort_key=&sort_direction=&page=. There is
// no LiveView session here, so the mount-time session values
// (initial_sort_key: :custom_name, initial_sort_direction: :asc,
// results_per_page: 10) become this handler's defaults for blank query
// params, and feed_base_url is recomputed from the request on every render
// instead of being fixed at mount.

import (
	"context"
	"net/http"
	"regexp"
	"strconv"

	sq "github.com/Masterminds/squirrel"
	"github.com/a-h/templ"
	"github.com/mattbriancon/pinchflat/internal/core"
)

const sourceLiveIndexTableLimit = 10

// sourceLiveIndexRow is the map(s, ^Source.__schema__(:fields)) |> select_merge
// shape from sources_query/0: a fixed, known result shape, so it's its own
// small struct rather than *core.Source (see CONVENTIONS "Types").
type sourceLiveIndexRow struct {
	ID               int64   `db:"id"`
	CustomName       string  `db:"custom_name"`
	UUID             *string `db:"uuid"`
	MediaProfileID   int64   `db:"media_profile_id"`
	MediaProfileName string  `db:"media_profile_name"`
	Enabled          bool    `db:"enabled"`
	DownloadedCount  int     `db:"downloaded_count"`
	PendingCount     int     `db:"pending_count"`
	MediaSizeBytes   int64   `db:"media_size_bytes"`
}

// sourceLiveIndexTableLiveState is this render's assigns.
type sourceLiveIndexTableLiveState struct {
	Sources       []*sourceLiveIndexRow
	SortKey       string
	SortDirection string
	Page          int
	TotalPages    int
	FeedBaseURL   string
}

// SourceLiveIndexTableLiveRender is PinchflatWeb.Sources.SourceLive.IndexTableLive.
func (s *Server) SourceLiveIndexTableLiveRender(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sortKey := sourceLiveIndexSortKey(r.URL.Query().Get("sort_key"))
	sortDirection := sourceLiveIndexSortDirection(r.URL.Query().Get("sort_direction"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	query := sourceLiveIndexTableLiveQuery()
	pag, err := s.GetPaginationAttributes(ctx, query, page, sourceLiveIndexTableLimit)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	ordered := query.OrderBy(sourceLiveIndexOrderBy(sortKey, sortDirection)).
		Limit(uint64(pag.Limit)).Offset(uint64(pag.Offset))
	sources, err := core.All[sourceLiveIndexRow](ctx, s.App.Q(ctx), ordered)
	if err != nil {
		s.Fail(w, r, err)
		return
	}

	state := &sourceLiveIndexTableLiveState{
		Sources:       sources,
		SortKey:       sortKey,
		SortDirection: sortDirection,
		Page:          pag.Page,
		TotalPages:    pag.TotalPages,
		FeedBaseURL:   PageOf(ctx).BaseURL + P(ctx, "/sources"),
	}
	s.RenderFragment(w, r, http.StatusOK, SourceLiveIndexTableLiveRenderComponent(state))
}

// sources_query/0: sources joined to their (non-deleted) media profile, with
// downloaded/pending counts and downloaded media size from subqueries.
func sourceLiveIndexTableLiveQuery() sq.SelectBuilder {
	downloadedSQL, downloadedArgs, _ := core.SQ.Select(
		"COUNT(mi.id) AS downloaded_count",
		"mi.source_id AS source_id",
		"SUM(mi.media_size_bytes) AS media_size_bytes",
	).From("media_items mi").Where(core.MediaQueryDownloaded()).GroupBy("mi.source_id").ToSql()

	pendingSQL, pendingArgs, _ := core.SQ.Select(
		"COUNT(mi.id) AS pending_count",
		"mi.source_id AS source_id",
	).From("media_items mi").
		Join("sources AS source ON source.id = mi.source_id").
		Join("media_profiles AS media_profile ON media_profile.id = source.media_profile_id").
		Where(core.MediaQueryPending()).
		GroupBy("mi.source_id").ToSql()

	q := core.SQ.Select(
		"s.id",
		"s.custom_name",
		"s.uuid",
		"s.media_profile_id",
		"media_profile.name AS media_profile_name",
		"s.enabled",
		"COALESCE(d.downloaded_count, 0) AS downloaded_count",
		"COALESCE(p.pending_count, 0) AS pending_count",
		"COALESCE(d.media_size_bytes, 0) AS media_size_bytes",
	).From("sources AS s").
		Join("media_profiles AS media_profile ON media_profile.id = s.media_profile_id").
		LeftJoin("("+downloadedSQL+") AS d ON d.source_id = s.id", downloadedArgs...).
		LeftJoin("("+pendingSQL+") AS p ON p.source_id = s.id", pendingArgs...).
		Where(sq.And{sq.Expr("s.marked_for_deletion_at IS NULL"), sq.Expr("media_profile.marked_for_deletion_at IS NULL")})

	return q
}

// sort_attr/1: the whitelisted, sortable column expressions.
func sourceLiveIndexSortAttr(sortKey string) string {
	switch sortKey {
	case "pending_count":
		return "p.pending_count"
	case "downloaded_count":
		return "d.downloaded_count"
	case "media_size_bytes":
		return "d.media_size_bytes"
	case "media_profile_name":
		return "media_profile.name COLLATE NOCASE"
	case "enabled":
		return "s.enabled"
	default:
		return "s.custom_name COLLATE NOCASE"
	}
}

func sourceLiveIndexOrderBy(sortKey, sortDirection string) string {
	dir := "ASC"
	if sortDirection == "desc" {
		dir = "DESC"
	}
	return sourceLiveIndexSortAttr(sortKey) + " " + dir + ", s.id ASC"
}

var sourceLiveIndexSortKeys = map[string]bool{
	"custom_name": true, "pending_count": true, "downloaded_count": true,
	"media_size_bytes": true, "media_profile_name": true, "enabled": true,
}

func sourceLiveIndexSortKey(v string) string {
	if sourceLiveIndexSortKeys[v] {
		return v
	}
	return "custom_name"
}

func sourceLiveIndexSortDirection(v string) string {
	if v == "asc" || v == "desc" {
		return v
	}
	return "asc"
}

// rss_feed_url/2.
func sourceLiveIndexRSSFeedURL(feedBaseURL string, row *sourceLiveIndexRow) string {
	return feedBaseURL + "/" + core.Deref(row.UUID) + "/feed.xml"
}

var sourceLiveIndexHTTPScheme = regexp.MustCompile(`^https?://`)

// podcast_app_url/1: the podcast:// scheme hands the feed to the device's
// podcast app (Apple Podcasts on iOS/macOS).
func sourceLiveIndexPodcastAppURL(feedURL string) string {
	return sourceLiveIndexHTTPScheme.ReplaceAllString(feedURL, "podcast://")
}

func sourceLiveIndexFragmentURL(ctx context.Context, sortKey, sortDirection string, page int) string {
	return P(ctx, "/_live/sources") + "?sort_key=" + sortKey + "&sort_direction=" + sortDirection + "&page=" + strconv.Itoa(page)
}

func sourceLiveIndexTargetAttrs(ctx context.Context, sortKey, sortDirection string, page int) templ.Attributes {
	return templ.Attributes{
		"hx-get":    sourceLiveIndexFragmentURL(ctx, sortKey, sortDirection, page),
		"hx-target": "#source-table",
		"hx-swap":   "outerHTML",
	}
}

// "sort_update": clicking a sortable column header.
func sourceLiveIndexHeaderAttrs(ctx context.Context, state *sourceLiveIndexTableLiveState) func(string) templ.Attributes {
	return func(colSortKey string) templ.Attributes {
		newDirection := GetSortDirection(state.SortKey, colSortKey, state.SortDirection)
		return sourceLiveIndexTargetAttrs(ctx, colSortKey, newDirection, 1)
	}
}

// "page_change".
func sourceLiveIndexPageAttrs(ctx context.Context, state *sourceLiveIndexTableLiveState, direction string) templ.Attributes {
	newPage := UpdatePageNumber(state.Page, direction, state.TotalPages)
	return sourceLiveIndexTargetAttrs(ctx, state.SortKey, state.SortDirection, newPage)
}
