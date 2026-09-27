package core

// Port of lib/pinchflat/profiles/profiles_query.ex (alias: media_profiles AS mp).

import sq "github.com/Masterminds/squirrel"

// new/0
func ProfilesQueryNew() sq.SelectBuilder { return From[MediaProfile]("mp") }
