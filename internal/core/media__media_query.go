package core

// Port of lib/pinchflat/media/media_query.ex. Hand-written in W0 because it
// defines the query-builder type the Media context and web tables compose.
//
// Table aliases match the Ecto binding names so fragments copy verbatim:
// media_items AS mi, sources AS source, media_profiles AS media_profile,
// and media_items_search_index (unaliased; FTS5 MATCH needs the table name).

import (
	"regexp"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattbriancon/pinchflat/internal/db"
)

// MediaQ is an Ecto query over media_items: a select builder plus the named
// bindings (joins) it already has, so RequireAssoc doesn't join twice.
// Methods return a new MediaQ; the receiver is not modified.
type MediaQ struct {
	B      sq.SelectBuilder
	joined map[string]bool
}

// new/0: every media_items column, aliased mi.
func MediaQueryNew() *MediaQ {
	return &MediaQ{B: From[MediaItem]("mi"), joined: map[string]bool{}}
}

func (q *MediaQ) with(b sq.SelectBuilder) *MediaQ {
	j := make(map[string]bool, len(q.joined))
	for k, v := range q.joined {
		j[k] = v
	}
	return &MediaQ{B: b, joined: j}
}

// Where adds a predicate (Ecto `where(^dynamic)`).
func (q *MediaQ) Where(pred sq.Sqlizer) *MediaQ { return q.with(q.B.Where(pred)) }

// Map applies arbitrary builder changes (order_by, limit, select, ...).
func (q *MediaQ) Map(fn func(sq.SelectBuilder) sq.SelectBuilder) *MediaQ { return q.with(fn(q.B)) }

// ToSql implements sq.Sqlizer.
func (q *MediaQ) ToSql() (string, []any, error) { return q.B.ToSql() }

// Not negates a predicate: `not (^pred)`.
func Not(pred sq.Sqlizer) sq.Sqlizer {
	return sqlizerFunc(func() (string, []any, error) {
		s, args, err := pred.ToSql()
		return "NOT (" + s + ")", args, err
	})
}

type sqlizerFunc func() (string, []any, error)

func (f sqlizerFunc) ToSql() (string, []any, error) { return f() }

// for_source/1 (Elixir accepts an id or a %Source{}; pass source.ID).
func MediaQueryForSource(sourceID int64) sq.Sqlizer { return sq.Eq{"mi.source_id": sourceID} }

// downloaded/0
func MediaQueryDownloaded() sq.Sqlizer { return sq.Expr("mi.media_filepath IS NOT NULL") }

// download_prevented/0
func MediaQueryDownloadPrevented() sq.Sqlizer { return sq.Expr("mi.prevent_download = 1") }

// culling_prevented/0
func MediaQueryCullingPrevented() sq.Sqlizer { return sq.Expr("mi.prevent_culling = 1") }

// redownloaded/0
func MediaQueryRedownloaded() sq.Sqlizer { return sq.Expr("mi.media_redownloaded_at IS NOT NULL") }

// upload_date_matches/1
func MediaQueryUploadDateMatches(otherDate time.Time) sq.Sqlizer {
	return sq.Expr("date(mi.uploaded_at) = date(?)", db.UTCDateTime{Time: otherDate})
}

// upload_date_after_source_cutoff/0 (needs the source binding)
func MediaQueryUploadDateAfterSourceCutoff() sq.Sqlizer {
	return sq.Expr("(source.download_cutoff_date IS NULL OR date(mi.uploaded_at) >= source.download_cutoff_date)")
}

// format_matching_profile_preference/0 (needs the media_profile binding)
func MediaQueryFormatMatchingProfilePreference() sq.Sqlizer {
	return sq.Expr(`CASE
          WHEN media_profile.shorts_behaviour = 'only' AND media_profile.livestream_behaviour = 'only' THEN
            mi.livestream = true OR mi.short_form_content = true
          WHEN media_profile.shorts_behaviour = 'only' THEN
            mi.short_form_content = true
          WHEN media_profile.livestream_behaviour = 'only' THEN
            mi.livestream = true
          WHEN media_profile.shorts_behaviour = 'exclude' AND media_profile.livestream_behaviour = 'exclude' THEN
            mi.short_form_content = false AND mi.livestream = false
          WHEN media_profile.shorts_behaviour = 'exclude' THEN
            mi.short_form_content = false
          WHEN media_profile.livestream_behaviour = 'exclude' THEN
            mi.livestream = false
          ELSE
            true
        END`)
}

// matches_source_title_regex/0 (needs the source binding)
func MediaQueryMatchesSourceTitleRegex() sq.Sqlizer {
	return sq.Expr("(source.title_filter_regex IS NULL OR regexp_like(mi.title, source.title_filter_regex))")
}

// meets_min_and_max_duration/0 (needs the source binding)
func MediaQueryMeetsMinAndMaxDuration() sq.Sqlizer {
	return sq.Expr("((source.min_duration_seconds IS NULL OR mi.duration_seconds >= source.min_duration_seconds) AND " +
		"(source.max_duration_seconds IS NULL OR mi.duration_seconds <= source.max_duration_seconds))")
}

// past_retention_period/0 (needs the source binding)
func MediaQueryPastRetentionPeriod() sq.Sqlizer {
	return sq.Expr(`IFNULL(source.retention_period_days, 0) > 0 AND
        mi.media_downloaded_at IS NOT NULL AND
        DATETIME(mi.media_downloaded_at, '+' || CAST(source.retention_period_days AS TEXT) || ' day') < DATETIME('now')`)
}

// past_redownload_delay/0 (needs the media_profile binding)
func MediaQueryPastRedownloadDelay() sq.Sqlizer {
	return sq.Expr(`IFNULL(media_profile.redownload_delay_days, 0) > 0 AND
        DATE('now', '-' || media_profile.redownload_delay_days || ' day') > DATE(mi.uploaded_at) AND
        DATE(mi.media_downloaded_at, '-' || media_profile.redownload_delay_days || ' day') < DATE(mi.uploaded_at)`)
}

// cullable/0
func MediaQueryCullable() sq.Sqlizer {
	return sq.And{MediaQueryDownloaded(), Not(MediaQueryCullingPrevented()), MediaQueryPastRetentionPeriod()}
}

// deletable_based_on_source_cutoff/0
func MediaQueryDeletableBasedOnSourceCutoff() sq.Sqlizer {
	return sq.And{MediaQueryDownloaded(), Not(MediaQueryUploadDateAfterSourceCutoff()), Not(MediaQueryCullingPrevented())}
}

// pending/0 (needs source and media_profile bindings)
func MediaQueryPending() sq.Sqlizer {
	return sq.And{
		Not(MediaQueryDownloaded()),
		Not(MediaQueryDownloadPrevented()),
		MediaQueryUploadDateAfterSourceCutoff(),
		MediaQueryFormatMatchingProfilePreference(),
		MediaQueryMatchesSourceTitleRegex(),
		MediaQueryMeetsMinAndMaxDuration(),
	}
}

// upgradeable/0 (needs the media_profile binding)
func MediaQueryUpgradeable() sq.Sqlizer {
	return sq.And{MediaQueryDownloaded(), Not(MediaQueryDownloadPrevented()), Not(MediaQueryRedownloaded()), MediaQueryPastRedownloadDelay()}
}

// require_assoc/2: joins the named binding unless already present.
// identifier is "source", "media_profile" or "media_items_search_index".
func (q *MediaQ) RequireAssoc(identifier string) *MediaQ {
	if q.joined[identifier] {
		return q
	}
	var out *MediaQ
	switch identifier {
	case "media_items_search_index":
		out = q.with(q.B.Join("media_items_search_index ON media_items_search_index.rowid = mi.id"))
	case "source":
		out = q.with(q.B.Join("sources AS source ON source.id = mi.source_id"))
	case "media_profile":
		q = q.RequireAssoc("source")
		out = q.with(q.B.Join("media_profiles AS media_profile ON media_profile.id = source.media_profile_id"))
	default:
		panic("require_assoc: unknown binding " + identifier)
	}
	out.joined[identifier] = true
	return out
}

// matching_search_term/2: filters by the FTS term, selects the highlighted
// snippet into matching_search_term and orders by rank.
func (q *MediaQ) MatchingSearchTerm(term *string) *MediaQ {
	if term == nil {
		return q
	}
	escaped := mediaQueryCleanSearchTerm(*term)
	q = q.RequireAssoc("media_items_search_index")
	return q.with(q.B.
		Where("media_items_search_index MATCH ?", escaped).
		Column(`coalesce(snippet(media_items_search_index, 0, '[PF_HIGHLIGHT]', '[/PF_HIGHLIGHT]', '...', 20), '') ||
            ' ' ||
            coalesce(snippet(media_items_search_index, 1, '[PF_HIGHLIGHT]', '[/PF_HIGHLIGHT]', '...', 20), '') AS matching_search_term`).
		OrderBy("rank DESC"))
}

var whitespaceRe = regexp.MustCompile(`\s+`)

// SQLite's FTS5 is picky about search terms: collapse whitespace, strip
// quotes and quote each word.
func mediaQueryCleanSearchTerm(term string) string {
	if term == "" {
		return ""
	}
	words := whitespaceRe.Split(strings.TrimSpace(whitespaceRe.ReplaceAllString(term, " ")), -1)
	for i, w := range words {
		words[i] = `"` + strings.ReplaceAll(w, `"`, "") + `"`
	}
	return strings.Join(words, " ")
}
