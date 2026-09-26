#!/usr/bin/env bash
# Regenerates the Elixir ground truth used by the Go compatibility tests.
# Requires Elixir/OTP (see docker/dev.Dockerfile for versions) and sqlite3.
set -euo pipefail
cd "$(dirname "$0")/../.."

out=testdata/elixir
db=priv/repo/pinchflat_test.db
export MIX_ENV=test

rm -f "$db" "$db"-*
mix ecto.create --quiet
mix ecto.migrate --quiet >/dev/null 2>&1
sqlite3 "$db" "VACUUM INTO '$out/schema.db'"

mix run "$out/capture.exs"
sqlite3 "$db" "VACUUM INTO '$out/populated.db'"
rm -f "$db" "$db"-*

# One line per column value: table|rowid|column|typeof|quote(value)
dump_values() {
  local file=$1
  for t in $(sqlite3 "$file" "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE 'media_items_search_index%' ORDER BY name"); do
    for c in $(sqlite3 "$file" "SELECT name FROM pragma_table_info('$t') ORDER BY cid"); do
      sqlite3 "$file" "SELECT '$t', rowid, '$c', typeof(\"$c\"), quote(\"$c\") FROM \"$t\" ORDER BY rowid"
    done
  done
}
dump_values "$out/populated.db" > "$out/golden/populated_values.txt"
sqlite3 "$out/schema.db" .schema > "$out/golden/schema.sql"
echo "wrote $out/schema.db, $out/populated.db and $out/golden/"
