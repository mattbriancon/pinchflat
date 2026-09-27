# Porting Pinchflat to Go — Strategy

> **Status:** Phase 1 done. Every Elixir test had a same-named Go test (970/970) before the Elixir tree was deleted (W5). The last commit with the Elixir app, `parity.py` and the capture tooling is `5ae7e94`. Later decisions are in §11.

Companion file: [`MANIFEST.md`](./MANIFEST.md) (one row per `lib/` file → Go target, wave, and test to port).

## 0. TL;DR

- **Hard constraint: an existing `/config/db/pinchflat.db` must work unchanged.** The Go binary opens it, picks up pending jobs, and keeps every URL that podcast apps have saved.
- **Cutover is one-way, with downtime.** Stop the Elixir container, start the Go one on the same volumes. Elixir and Go never run at the same time, and there is no rollback to the Elixir image after Go has written to the DB (restore the pre-cutover DB backup instead). So Go only needs to *read* what Elixir wrote; it doesn't need to stay readable by Elixir.
- **Data layer:** `modernc.org/sqlite` (pure Go, so no CGO or `.so` files), `sqlx` for scanning, and `squirrel` for building the dynamic queries that Ecto builds today. **No ORM.** A small set of hand-written types reproduce exactly how Ecto encodes values on disk.
- **Migrations:** a small in-house migrator that writes to **Ecto's own `schema_migrations` table**. It uses the same 79 version numbers, with each `.exs` file transcribed to a `.sql` file. It is verified by a golden test against a database migrated by the real Elixir app.
- **Jobs:** a small in-house job runner (`internal/obanlite`) that reads and writes the **existing `oban_jobs` table**, with the same column semantics and the same worker name strings. We don't use River or any other queue that brings its own schema, because `tasks.job_id` has a foreign key into `oban_jobs` and existing DBs contain pending jobs.
- **Web:** `chi` for routing, `templ` for templates (converted from the `.heex` files), htmx for form interactions (Alpine.js stays). No server push: live-updating tables become a refresh button or a page reload.
- **Process:** Phase 0 builds the compatibility harness and infrastructure, done by a stronger model. Phase 1 ports the code file by file in dependency waves, with **Haiku agents doing one file and its test each**, compiling against a pre-generated skeleton. Phase 2 comes after cutover and makes the code idiomatic Go.
- **Can be dropped:** `tooling/` entirely, most of `rel/`, and much of `priv/`. **Not all of `priv/`:** `priv/repo/migrations` defines the schema we must match, and `priv/static` holds the app's images and fonts. See §8.

---

## 1. What "backwards compatible" means here (the contract)

Every item below becomes an automated test in Phase 0 or a checklist item for cutover.

| Surface | Contract |
|---|---|
| SQLite schema | Identical tables, columns, declared types, defaults, indexes, FKs, FTS5 table and triggers. **No schema changes during Phase 1 or Phase 2.** |
| Stored value encoding | Go reads every value Ecto wrote. During Phase 1 Go also *writes* the same encodings (see §2.3), because the SQL compares timestamps as text and mixing formats would break those queries. Phase 2 may normalise encodings via a migration. |
| `schema_migrations` | Same table and same version numbers, so Go knows exactly which Elixir migrations an existing DB has applied. |
| `oban_jobs` / `tasks` | Jobs queued by Elixir before cutover (including every source's next scheduled index) run under Go. Worker names stay `Pinchflat.Downloading.MediaDownloadWorker` and so on. |
| URLs | Every route in `router.ex` stays byte-identical, including `/sources/:uuid/feed`, `/sources/opml?route_token=…`, `/media/:uuid/stream`, `…/feed_image`, `…/episode_image`, and `/healthcheck` (`{"status":"ok"}`). This includes the endpoint behaviour that **strips a trailing extension** (`endpoint.ex:85-115`) and the base URL built from `x-forwarded-proto`. |
| Env vars | Everything in `config/runtime.exs`: `MEDIA_PATH`, `CONFIG_PATH`, `DATABASE_PATH`, `LOG_PATH`, `METADATA_PATH`, `EXTRAS_PATH`, `TMPFILE_PATH`, `PORT`, `BASIC_AUTH_USERNAME/PASSWORD`, `EXPOSE_FEED_ENDPOINTS`, `BASE_ROUTE_PATH`, `ENABLE_IPV6`, `ENABLE_PROMETHEUS`, `YT_DLP_WORKER_CONCURRENCY`, `LOG_LEVEL`, `JOURNAL_MODE`, `SECRET_KEY_BASE`, `TZ`, and `UMASK` (from `docker_start`). `TZ_DATA_PATH`, `DNS_CLUSTER_QUERY` and `PHX_SERVER` can be ignored. |
| Filesystem layout | Same default paths. `extras/cookies.txt`, `extras/user-scripts/lifecycle`, the yt-dlp config under `/etc/yt-dlp`, and the metadata directory layout all stay the same. |
| User script hook | `lifecycle <event> <json>` with events `app_init`, `media_pre_download`, `media_downloaded`, `media_deleted`. The JSON shape comes from the **custom `Jason.Encoder` impls** in `media_item.ex:190`, `source.ex:186` and `media_profile.ex:101`, so field names and field sets must match. |
| Apprise | Removed (§11.5). Its settings columns are kept, unused. |
| yt-dlp invocations | Same argv for the same inputs. The option builder tests guarantee this. |
| RSS/OPML XML | Byte-for-byte identical output for the same DB. Verified by a diff test against the Elixir output. |
| Docker image | Same volumes (`/config`, `/downloads`, `/etc/yt-dlp`), same port env, same healthcheck, same PUID/UMASK behaviour, same bundled tools (yt-dlp, ffmpeg, deno, apprise). |
| `/metrics` | **Names change, endpoint stays.** Same path, same `ENABLE_PROMETHEUS` switch, Prometheus text format, new metric names designed for Go (see §4.6). Scrapeable by the Datadog agent's OpenMetrics check. |

---

## 2. Database: driver, ORM and migrations

### 2.1 Driver: `modernc.org/sqlite`

- Pure Go, so cross-compiling the `linux/amd64` and `linux/arm64` Docker images needs no CGO toolchain.
- Ships FTS5 with the **`trigram` tokenizer**, which `media_items_search_index` requires, along with `snippet()` and `rank`.
- Lets us register Go scalar functions. We need that because the app loads **sqlean** (`config/runtime.exs:39`) only to get `regexp_like()`, which `media_query.ex:74` and `source.ex:167` use. We register our own `regexp_like` instead of shipping `.so` files.
  - ⚠️ **Regex flavour trap.** sqlean's `regexp_like` is PCRE-flavoured. Go's `regexp` is RE2 and rejects lookarounds and backreferences, so saved `title_filter_regex` values could stop matching or fail validation. Use `github.com/dlclark/regexp2` for this function, and add a test corpus of PCRE-style patterns.
- Alternatives considered: `mattn/go-sqlite3` needs CGO and the `sqlite_fts5` build tag; `ncruces/go-sqlite3` (Wasm) is viable. Revisit only if modernc performance is a problem.

**Connection setup must match ecto_sqlite3 0.19 defaults.** Apply these per connection through the DSN or a connect hook:
`journal_mode=WAL` (or `JOURNAL_MODE`), `foreign_keys=ON` (**required**: `tasks → oban_jobs ON DELETE CASCADE` and the metadata cascades depend on it), `busy_timeout` of about 2000ms or more, `synchronous=NORMAL`, `temp_store=MEMORY`. Use one writer connection plus a small reader pool to avoid `SQLITE_BUSY` storms, since Elixir used `pool_size: 5` with its own queueing.

### 2.2 Query layer: `sqlx` + `squirrel`, no ORM

- The difficult query code, `media_query.ex`, is **composable**. It combines `dynamic()` predicates such as `pending`, `cullable`, `past_retention_period`, `format_matching_profile_preference` and `matches_source_title_regex` with raw `fragment()` SQL (`CASE` expressions, `DATE(… '+' || n || ' day')`, `MATCH`, `snippet()`, `COLLATE NOCASE`). `squirrel`'s `sq.And{}`/`sq.Expr()` maps almost 1:1 onto Ecto `dynamic` + `fragment`, which makes the file-by-file port mechanical.
- `sqlx` scans rows into plain structs that mirror each Ecto schema.
- **Rejected:**
  - **GORM:** AutoMigrate is a foot-gun, it has its own `created_at` naming and soft-delete conventions, and the driver serialises times itself (breaks §2.3).
  - **ent:** it owns the schema and migrations.
  - **sqlc:** it can't express the dynamic query composition. It is a good fit for the static queries in **Phase 2**.
  - **bun:** workable, but it adds conventions we would have to fight.
- Preloads (`Repo.preload`) become explicit follow-up queries or `JOIN`s in the context functions.
- Upserts (`media.ex:140-148`, conflict target `[:source_id, :media_id]`, `playlist_index` excluded from the update) become `INSERT … ON CONFLICT(source_id, media_id) DO UPDATE SET …`.
- **Changesets** become a small `Changeset`-like helper in `internal/core/utils__changeset_utils.go`: cast, validate required/format/number, collect field errors, and translate unique-constraint errors. Error message strings must stay identical, because controller tests assert on them. This is a porting shim; Phase 2 may simplify it.

### 2.3 Ecto encoding compatibility (the #1 risk)

Build `internal/db/ectotypes` with `Scanner`/`Valuer` implementations and **never let the driver serialise `time.Time` itself**. The driver's default format is not Ecto's, and the SQL uses `DATE()`/`DATETIME()` and compares timestamp strings as text.

| Ecto type | On-disk form (**confirm in Phase 0 by dumping a real DB**) | Go type |
|---|---|---|
| `:utc_datetime` (all `timestamps()`, `*_at`) | `TEXT` `2024-03-01T12:34:56Z` (second precision) | `ectotypes.UTCDateTime` |
| `media_items.uploaded_at` | **Mixed**: migration `20240528180212` backfilled `YYYY-MM-DDT00:00:00` with **no `Z`** and set that as the column default; Ecto writes `…Z`. Reads must accept both; writes use Ecto's form. | same type, lenient parse |
| `:date` (`download_cutoff_date`) | `TEXT` `YYYY-MM-DD` | `ectotypes.Date` |
| `:boolean` | `INTEGER` 0/1 | `bool` |
| `{:array, _}` (`subtitle_filepaths` is `[[lang, path], …]`; `sponsorblock_categories`) | JSON `TEXT` | `ectotypes.JSON[T]` |
| `Ecto.Enum` | `TEXT` atom name (`"only"`, `"1080p"`, `"when_needed"`) | named `string` types plus a validation list |
| `uuid` | lowercase hyphenated `TEXT` | `string` |

Oban columns (`args`, `meta`, `tags`, `errors`, `attempted_by`) are JSON text written by the Oban Lite engine and need the same treatment.

### 2.4 Migrations: reuse Ecto's `schema_migrations` table

Off-the-shelf tools collide with it. `golang-migrate` uses a table **also named `schema_migrations`** but with a different shape (`version, dirty`), and `goose` uses its own table. So we write a ~150-line migrator (`internal/db/migrate`):

1. Create `schema_migrations(version INTEGER PRIMARY KEY, inserted_at TEXT)` if it is missing, in Ecto's exact DDL.
2. Embed `migrations/<14-digit version>_<name>.sql`, **one per existing `.exs`, with the same version numbers**, and apply the ones missing from the table inside a transaction. Insert the version in Ecto's `inserted_at` format.
3. Fresh installs run all 79 migrations. Old installs run only the missing ones. That means one code path, and users on old Elixir releases can jump straight to Go.
4. Oban's `Oban.Migration.up(version: 11)` (`20240125043813`) becomes literal SQL taken from Oban 2.19.4's SQLite migration.
5. Data migrations need care: `backfill_content_uuids` (uses sqlean's `gen_random_uuid()`, which Go must also register or replace), `create_new_settings` (seeds the single settings row), `rename_upload_date_to_uploaded_at`, `add_route_token_to_settings` (backfills a random token), and `add_cookie_behaviour_to_sources`.
6. After cutover, new migrations use new 14-digit versions in the same table. They don't need to stay compatible with Elixir, because there is no rollback.

**Golden test (the gate for all of this).** In CI, use the existing `docker/dev.Dockerfile` to run `mix ecto.migrate` on an empty DB and commit the result as `testdata/elixir_schema.db`. The Go test runs its migrator on an empty DB and compares both databases through `sqlite_master` SQL text plus `PRAGMA table_info / index_list / index_xinfo / foreign_key_list` for every table. Declared type strings must match exactly, because they drive SQLite type affinity. Use ecto_sqlite3's type names, not our own.

---

## 3. Background jobs: `internal/obanlite`

Written in Phase 0 by the stronger model; roughly 600–900 LOC plus tests.

**Why not River or asynq:** existing DBs contain `available`/`scheduled`/`retryable` rows in `oban_jobs`, `tasks.job_id` has a foreign key to `oban_jobs(id)` with `ON DELETE CASCADE`, and the dashboard (`job_table_live.ex`) reads `oban_jobs` columns directly. Every source has its next index run sitting in that table as a `scheduled` job, so dropping it at cutover would silently stop indexing. Keeping the table is less work than converting it. If a different queue is ever wanted, Phase 2 can move to it with a one-off data migration.

Features to replicate (read the Oban 2.19.4 Lite engine source; don't guess):

- **Queues and limits:** `default` 10, `local_data` 8, and `fast_indexing` / `media_collection_indexing` / `media_fetching` / `remote_metadata` each at `YT_DLP_WORKER_CONCURRENCY` (default 2).
- **Worker registry keyed by Oban's string names.** Registering all 10 workers under their exact names means jobs Elixir queued before cutover run under Go.
- **Fetch:** `BEGIN IMMEDIATE`, select available jobs by `priority, scheduled_at, id`, then mark them `executing`, increment `attempt`, and set `attempted_at` and `attempted_by`.
- **Staging:** move `scheduled`/`retryable` jobs to `available` once they are due.
- **Uniqueness:** each worker's `unique:` options (`period: :infinity`, states list, default keys worker+queue+args) are checked with a query at insert time, as the Lite engine does, and report `conflict?`. `Repo.insert_unique_job` depends on that result.
- **Retries and failures:** `max_attempts` (default 20; `SourceMetadataStorageWorker` uses 3), Oban's default exponential backoff with jitter, `errors` appended as `{at, attempt, error}`, and `discarded` when attempts run out.
- **Cancel:** the state becomes `cancelled`, and if the job is executing, its `context` is cancelled. This replaces `priv/cmd_wrapper.sh` (see §5).
- **Cron:** yt-dlp update at a per-instance staggered time, retention at 01:00 UTC, quality upgrade at 02:00 UTC. Match how Oban's Cron plugin writes `meta` and handles uniqueness.
- **Pruner:** delete finished jobs after 30 days. The FK cascade deletes their `tasks` rows, so `foreign_keys=ON` is required.
- **Boot rescue:** `PreJobStartupTasks` resets `executing` jobs to `retryable`.
- **Events:** none needed for the UI. The `job:state` and `media_table` `Phoenix.PubSub` topics only drive LiveView refreshes, which are dropped (§4.5). Job start/stop hooks exist only to feed metrics.
- **Test mode:** `obanlite.Manual` plus `AssertEnqueued(t, worker, args)` / `RefuteEnqueued`, mirroring `Oban.Testing` so worker tests port 1:1.

---

## 4. Code layout and the file-by-file mechanics

### 4.1 Layout (Phase 1)

```
go.mod                      (repo root, alongside mix.exs until cutover)
cmd/pinchflat/main.go       (application.ex, docker_start, check_file_permissions, migrate)
internal/db/                (connection, pragmas, ectotypes, migrate/, regexp_like)   ← hand-written, W0
internal/obanlite/          (job runner)                                              ← hand-written, W0
internal/core/              (ALL of lib/pinchflat/** as ONE flat package)
internal/web/               (ALL of lib/pinchflat_web/** as ONE flat package, *.templ + *.go)
internal/testsupport/       (DataCase/ConnCase equivalents, fixtures, mocks)
migrations/*.sql            (79 files, same versions)
web/static/                 (from priv/static, go:embed)
```

**Why one flat `core` package:** the Elixir contexts call each other in cycles. For example, `Sources` → `SlowIndexingHelpers` → `Sources`, and `Sources` ↔ `Media` ↔ `Tasks` → workers → `Sources`. Go forbids import cycles, so splitting into packages now would force redesign during what should be a mechanical port. File names encode the original path (`lib/pinchflat/downloading/media_downloader.ex` → `internal/core/downloading__media_downloader.go`) so the mapping stays greppable. Phase 2 breaks the package up.

### 4.2 Porting conventions (`docs/go-port/CONVENTIONS.md`, written in Phase 0; every Haiku prompt includes it)

- Elixir module → a Go type or function prefix: `Sources.get_source!/1` → `SourcesGetSource(ctx, id) (*Source, error)`. Phase 2 renames.
- `{:ok, x} | {:error, e}` → `(x, error)`. Bang functions → the same function returning an error; no panics except where Elixir would crash the process on purpose.
- Behaviours (`YtDlpCommandRunner`, `HTTPBehaviour`, `YoutubeBehaviour`, `AppriseCommandRunner`, `UserScriptCommandRunner`) → Go interfaces on a `Deps` struct. The four Mox mocks become hand-written fakes with an expectation helper in `testsupport`.
- `Application.get_env` → a `Config` struct loaded once from env vars in `main.go`.
- Each Elixir `test "…"` → a `t.Run("…")` with **the same name string**, so parity can be checked by script.
- Never change a signature from the skeleton. If one is wrong, stop and report back instead of editing other files.

### 4.3 Skeleton-first, so Haiku agents can work in parallel

The cycles would otherwise force a strict serial order, so we split the port into two steps:

1. **Skeleton pass (W0, Haiku, one agent per directory, reviewed by Sonnet/Opus).** For every row in `MANIFEST.md`, emit the Go file with the structs and every public function signature, with bodies set to `panic("unported: <module>.<fn>")`. The result is `go build ./...` passing on the whole tree.
2. **Fill pass (W1–W4).** One Haiku agent per manifest row ports the function bodies and the matching Elixir test file. The whole tree compiles at every step, so agents never block each other.

### 4.4 Waves

Counts come from `MANIFEST.md`.

| Wave | Contents | Who | Exit gate |
|---|---|---|---|
| **W0** | Ground-truth capture (§6.1), `internal/db` (types, migrator, 79 `.sql` files), `internal/obanlite`, `testsupport`, `CONVENTIONS.md`, skeleton, CI job | Opus/Sonnet writes infra; **Haiku** transcribes the 79 migrations (batches of ~10) and generates the skeleton | Schema golden test and value round-trip test green; `go build ./...` green |
| **W1** (29 files) | Pure logic: utils, output path parser (nimble_parsec → a hand-written recursive-descent parser for `{{ var }}` templates), download and quality option builders, NFO/RSS/OPML builders, metadata parsers, yt-dlp runners, HTTP client, file follower (GenServer → goroutine tailing a file) | **Haiku**, 8–10 in parallel | Ported tests pass; argv/XML byte-equality tests pass |
| **W2** (17 files) | Schemas, `*_query.ex`, contexts (media, sources, profiles, settings, tasks, metadata records) | **Haiku**; `media_query.ex` and `media.ex` get a Sonnet review | Context tests pass against a temp copy of the golden DB |
| **W3** (19 files) | Workers, helpers, boot tasks, lifecycle notifications, `main.go` | **Haiku**; `main.go` and boot by Sonnet | Worker tests pass using `obanlite` manual mode; the app boots against `elixir_populated.db` and drains a queued job |
| **W4** (69 files) | Router, plugs, endpoint middleware, controllers, 36 `.heex` → `.templ`, components, LiveViews → handlers + htmx, metrics, static embed | **Haiku** for templates and controllers; Sonnet for router and plugs | Controller tests pass; route-table parity; RSS/OPML diff vs Elixir |
| **W5** | New `docker/selfhosted.Dockerfile` (Go build stage + same runtime tools), CI workflows, release notes, cutover PR that deletes the Elixir tree | Sonnet | §6 compatibility suite green; manual QA checklist |

Merge one PR per wave (or two for W4) into the branch. The Elixir app keeps shipping from `master` until W5. W5 ships as a single release: back up the DB, stop Elixir, start Go on the same volumes.

**Cutover runbook (W5).** 1) Stop the Elixir container. 2) Copy `/config/db/pinchflat.db*` (including `-wal`/`-shm`) aside. 3) Start the Go image on the same volumes; it checks `schema_migrations`, applies nothing (or any missing Elixir-era migrations), and resumes queued jobs. 4) Smoke-check `/healthcheck`, the sources page, a feed and the job table. 5) If anything is wrong, stop Go, restore the DB copy, and start the old Elixir image.

### 4.5 Web specifics

- **Templates:** `.heex` function components map closely onto `templ` components (typed params, slots → `templ.Component` children). Keep Tailwind classes verbatim and point `tailwind.config.js` content globs at `*.templ`. Use the standalone Tailwind CLI and drop esbuild: the JS is Alpine plus two small helper files, which can be vendored and served as-is.
- **LiveViews:** none need server push. `upgrade_button_live`, `apprise_server_live`, `source_enable_toggle` and `index_table_live` are forms, debounced search and toggles, so htmx covers them (`hx-trigger="keyup changed delay:200ms"`, etc.).
  - `history_table_live` and `media_item_table_live` already refresh only when the user clicks their refresh button (`reload_page`). That becomes a plain link or an `hx-get` for the table fragment. The `media_table` broadcast, which refreshes every table on a source page at once, becomes a full page reload.
  - `job_table_live` (home page, running jobs) is the only view that updates itself today, via the `job:state` topic. It becomes a static table with the same refresh button; reloading the page shows current state.
- **Forms:** keep Phoenix's param names (`source[custom_name]`) so templates and tests port directly; a small decoder strips the prefix.
- **CSRF:** `net/http.CrossOriginProtection` (Go 1.25+). **Flash:** a signed cookie keyed from `SECRET_KEY_BASE`.
- **Streaming:** `http.ServeContent` handles Range/206 natively, which replaces the hand-rolled range parsing in `media_item_controller.ex`.
- **Auth:** basic auth when both env vars are set; `maybe_basic_auth` skipped when `EXPOSE_FEED_ENDPOINTS` is set; `token_protected_route` compares `route_token` to `settings.route_token`. `BASE_ROUTE_PATH` becomes a `chi` mount prefix.

### 4.6 Metrics (`/metrics`)

`prometheus/client_golang`, served on `/metrics` only when `ENABLE_PROMETHEUS` is set, outside basic auth (the current PromEx plug also sits in front of the router). Metrics:

- **Runtime:** the standard Go and process collectors (memory, goroutines, GC, CPU, open FDs).
- **HTTP:** `pinchflat_http_requests_total{route,method,status}`, `pinchflat_http_request_duration_seconds{route,method}`. Label by route pattern, not raw path, to keep cardinality low.
- **Jobs:** `pinchflat_jobs_total{queue,worker,outcome}` (completed/retryable/discarded/cancelled), `pinchflat_job_duration_seconds{queue,worker}`, and gauges `pinchflat_jobs{queue,state}` sampled from `oban_jobs`.
- **Domain:** gauges for sources (enabled/disabled), media items (downloaded/pending/culled), bytes on disk, plus `pinchflat_ytdlp_exit_total{code}`.
- **DB:** `sql.DBStats` via the client_golang DB collector.

Datadog agent config (`conf.d/openmetrics.d/conf.yaml`):

```yaml
instances:
  - openmetrics_endpoint: http://pinchflat:8945/metrics
    namespace: pinchflat
    metrics: [".*"]
```

---

## 5. External processes

- `System.cmd` through `priv/cmd_wrapper.sh` exists only so that cancelling a job kills yt-dlp; the wrapper kills the child when stdin closes. In Go, use `exec.CommandContext` with `SysProcAttr{Setpgid: true}` and a `Cancel` func that kills the **process group**, because yt-dlp spawns ffmpeg. The wrapper script can then be deleted.
- Keep running commands with the working directory set to the tmpfile dir, accept exit codes 0 and 101, capture output to a temp file, and suppress logging of the user-script JSON payload.
- The shell mocks in `test/support/scripts/yt-dlp-mocks/*.sh` are reused unchanged by the Go tests.

---

## 6. Compatibility test suite

Built in W0 and required on every PR.

1. **Ground-truth capture** (a one-off script, re-runnable in CI via `docker/dev.Dockerfile`):
   - `testdata/elixir_schema.db`: freshly migrated.
   - `testdata/elixir_populated.db`: created by an Elixir script that uses the existing fixture modules to insert profiles, sources (both collection types, with a regex filter, a cutoff date, and every `cookie_behaviour`), media items (subtitles array, legacy no-`Z` `uploaded_at`, FTS content), metadata rows, settings, and jobs in every Oban state with linked tasks.
   - `testdata/elixir_values.txt`: `SELECT typeof(c), quote(c)` for every column of every table, to pin down on-disk encodings.
   - Golden outputs: RSS for two sources, OPML, `/healthcheck`, the user-script JSON for a media item, source and profile, and the yt-dlp argv for a set of profiles.
2. **Schema golden** (§2.4).
3. **Round-trip identity:** load every row of the populated DB into Go structs, save them back unchanged, and require every column to be byte-identical (`quote(c)` matches).
4. **Queued-job pickup:** Go runs every Elixir-enqueued job in the populated DB (all 10 workers, all states), including the uniqueness checks against existing rows.
5. **Cutover rehearsal (W5):** start the Go binary on a copy of the populated DB with the mock yt-dlp, let it index, download and prune, and check `/healthcheck`, `/sources`, a feed, the job table and `/metrics`. Then repeat on a copy of your real production DB.
6. **HTTP parity:** walk the `router.ex` route list and assert each route exists in Go with the same method and path; diff feed output.
7. **Test-count parity:** a script counts Elixir `test "` names per file against Go `t.Run` names from the manifest.

---

## 7. Phase 2: idiomatic Go (after the Phase 1 cutover is merged)

Guardrails: the §6 suite doesn't change and must stay green. Schema changes are now allowed, but only through new migrations with a test on the populated fixture DB (for example normalising the mixed `uploaded_at` format). Haiku does the per-package refactors; Sonnet designs the package boundaries and reviews.

- Split `internal/core` into packages along real dependency lines, such as `store` (repositories), `jobs`, `ytdlp`, `indexing`, `downloading`, `notify`, `feeds`, `metadata`. Break cycles with small interfaces and by enqueuing jobs by name (already cycle-free thanks to the string worker registry).
- Rename to Go conventions (`SourcesGetSource` → `store.Sources.Get`), pass `context.Context` everywhere, use `slog` for logging, wrap errors with `%w`, and collapse the tuple-shaped returns.
- Replace the changeset shim with plain validation methods returning `map[string][]string`.
- Move the static queries to `sqlc`; keep `squirrel` only for the `media_query` composition.
- Consolidate tests into table-driven form, use `testing/synctest` for the file follower and scheduler tests, and add `t.Parallel()` where the per-test DB copy allows it.
- Consider replacing `obanlite` polling with a write-notify channel for lower latency, keeping the table format.

---

## 8. What to drop, keep or port

Your instinct is mostly right. The exceptions are **`priv/repo/migrations` and `priv/static`**, which must be kept or ported, and the three scripts in `rel/overlays/bin`, whose *logic* has to be ported even though the files go.

| Path | Verdict | Why |
|---|---|---|
| `tooling/` (`.credo.exs`, `.check.exs`, `version_bump.sh`) | **Drop** | Elixir lint/check configs. Replace the version bump with `-ldflags "-X main.version=…"` in the Dockerfile. |
| `rel/overlays/bin/migrate`, `lib/pinchflat/release.ex` | **Drop** | The migrator runs inside the Go binary at startup (plus an optional `pinchflat migrate` subcommand). |
| `rel/overlays/bin/docker_start` | **Drop the file, port the logic** | Permission check → `umask $UMASK` → migrate → start becomes `main.go` using `syscall.Umask`. |
| `rel/overlays/bin/check_file_permissions` | **Drop the file, port the logic** | The user-facing error messages ("chown 99:100 …") move into Go boot. |
| `priv/repo/migrations/*.exs` | **Keep until cutover, then delete** | These define the schema we must match and are the source for the 79 `.sql` files. Delete after W5 once the golden DB is committed. |
| `priv/static/` | **Keep → move to `web/static/`** | Favicon, logos, Satoshi font, robots.txt; served via `go:embed`. |
| `priv/cmd_wrapper.sh` | **Drop** | Replaced by process-group kill (§5). |
| `priv/repo/extensions/sqlean-*` | **Drop** | Binary `.so` files used only for `regexp_like` (and `gen_random_uuid` in an old migration); replaced by Go-registered functions. |
| `priv/grafana/*.json` | **Drop** | BEAM, Phoenix, Ecto, Oban and LiveView dashboards; meaningless for Go. |
| `priv/gettext/`, `lib/pinchflat_web/gettext.ex` | **Drop** | Only `errors.pot` plus three `gettext("…")` calls in `core_components.ex`; inline the English. |
| `priv/repo/seeds.exs` | **Drop** | Empty. |
| `priv/repo/erd.png`, root `package.json` (`sqleton`, `prettier`) | **Drop** | The ERD is generated from the dev DB. Keep prettier only if you still want it for `.templ`/JS/CSS. |
| `lib/pinchflat/mailer.ex` + Swoosh | **Drop** | Never called. |
| `lib/pinchflat_web/controllers/sources/source_html/index_table_live.ex` | **Drop** | Dead code: `PinchflatWeb.Sources.IndexTableLive` is never referenced; the live one is `SourceLive.IndexTableLive`. |
| `lib/pinchflat/prom_ex.ex`, `lib/pinchflat_web/telemetry.ex`, LiveDashboard (`/dev/dashboard`), `dns_cluster` | **Drop** | Replaced by `prometheus/client_golang` on `/metrics` behind `ENABLE_PROMETHEUS`. |
| `lib/pinchflat.ex`, `lib/pinchflat_web.ex`, `post_job_startup_tasks.ex`, `error_json.ex`, the `*_behaviour.ex` modules | **Drop** | Empty or macro-only modules; behaviours become interfaces next to their implementations. |
| `mix.exs`, `mix.lock`, `config/*.exs`, `.formatter.exs`, `assets/package.json` + esbuild, `docker-compose.ci.yml`, `docker/dev.Dockerfile` | **Drop at W5** | Keep until cutover. `dev.Dockerfile` is also how CI regenerates the golden DBs. |
| `test/support/files/*`, `test/support/scripts/yt-dlp-mocks/*` | **Keep → `testdata/`** | Reused verbatim by the Go tests. |
| `docker/selfhosted.Dockerfile` | **Rewrite** | Same runtime layer (yt-dlp, ffmpeg, deno, apprise, `ENV`, `VOLUME`, `HEALTHCHECK`); the build stage becomes `golang` plus the Tailwind CLI. |

---

## 9. Haiku agent playbook

**Model routing.**
- **Haiku:** per-file ports and test ports, migration SQL transcription, `.heex` → `.templ` conversion, skeleton generation, surveys, parity scripts.
- **Sonnet/Opus:** `internal/db`, `internal/obanlite`, `CONVENTIONS.md`, skeleton review, `media_query`/`router`/`main.go`, per-wave review, and any file where Haiku fails twice.

**Task packet** (identical for every file):
1. Source file path.
2. Test file path (from `MANIFEST.md`).
3. Target Go path.
4. `CONVENTIONS.md`.
5. The skeleton file(s) it touches.
6. Acceptance: `go vet ./... && go build ./... && go test ./internal/core -run '<TestPrefix>'` pass, and test-name parity passes.
7. Rules: "don't edit other files; report back if a signature must change."

**Concurrency.** 8–10 Haiku agents per batch, each in its own git worktree (`isolation: "worktree"`), merged sequentially by the orchestrator. Conflicts can't happen because each agent owns exactly one target file plus its `_test.go`.

**Escalation.** A second failure on the same file goes to Sonnet with both attempts' diffs. Any agent report asking to change a skeleton signature goes to the orchestrator, which updates the skeleton and re-runs dependants.

**Review.** Per wave, one Sonnet pass reads the whole diff for cross-file semantic drift (error strings, time handling, JSON field names) against the §1 contract.

---

## 10. Risk register

| Risk | Mitigation |
|---|---|
| Timestamp format drift → broken retention, redownload and cutoff queries | `ectotypes`; round-trip test; mixed `uploaded_at` fixture |
| PCRE vs RE2 regex semantics | `regexp2` plus pattern corpus |
| FK cascades silently off | Assert `PRAGMA foreign_keys` = 1 in a test; cascade test via the pruner |
| Oban uniqueness/backoff subtly different → duplicate downloads or hot retries | Port against the Oban 2.19.4 source; queued-job pickup test; unique-insert tests ported from worker tests |
| Orphaned yt-dlp/ffmpeg on cancel | Process-group kill test with a mock script that forks |
| SQLite write contention (Go concurrency > BEAM pool of 5) | Single writer connection, `BEGIN IMMEDIATE`, busy_timeout |
| User-script JSON shape drift | Golden JSON from the Jason encoders |
| Podcast clients breaking | Extension-stripping middleware, URL parity test, RSS byte-diff |
| LiveView UX regressions (job table no longer updates itself) | Accepted; refresh button + page reload; manual QA checklist in W5 |
| `/metrics` consumers | Names change (accepted); endpoint and switch unchanged; Datadog config in §4.6 |
| Cutover goes wrong | DB backup step in the runbook; rehearsal on a copy of the production DB |

## 11. Decisions

1. **Cutover:** decided. Hard cutover with downtime; Elixir and Go never coexist; no rollback to Elixir except by restoring the DB backup.
2. **`/metrics`:** decided. Names may change; the endpoint must keep working, and it's scraped by the Datadog agent (§4.6).
3. **Repo:** decided. Same repo, `go.mod` at the root, Elixir deleted in W5.
4. **Frontend:** decided, then revised after the port: server-rendered `templ` + Alpine.js + Tailwind, **no htmx**. Every page renders everything on first load; anything new (paging, search, sorting, refresh) is a normal link or form that reloads the page. Toggles and buttons are plain form POSTs that redirect back.
5. **Apprise:** removed after the port. No notifications are sent and the image no longer bundles apprise/pipx. The `settings.apprise_server` and `apprise_version` columns stay in the database untouched (no migration drops them), so the schema stays what Elixir left.
