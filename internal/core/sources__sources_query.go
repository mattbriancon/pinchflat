package core

// Port of lib/pinchflat/sources/sources_query.ex (alias: sources AS s).

import sq "github.com/Masterminds/squirrel"

// new/0
func SourcesQueryNew() sq.SelectBuilder { return From[Source]("s") }

// for_media_profile/1 (Elixir accepts an id or a %MediaProfile{}; pass the id).
func SourcesQueryForMediaProfile(mediaProfileID int64) sq.Sqlizer {
	return sq.Eq{"s.media_profile_id": mediaProfileID}
}
