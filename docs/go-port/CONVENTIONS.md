# Go port conventions (Phase 1: faithful port)

Every agent porting a file reads this first. The goal of Phase 1 is a **mechanical, faithful** port: same behaviour, same error strings, same SQL, same test cases. Idiomatic cleanup happens in Phase 2; don't do it now.

## Where things go

- `lib/pinchflat/**/*.ex` → `internal/core/<path with / → __>.go`, all in **one package `core`**. The exact target file for each source file is in `docs/go-port/MANIFEST.md`.
- Tests `test/pinchflat/**/*_test.exs` → `internal/core/<same name as the target>_test.go`, **`package core_test`**.
- `test/support/**` → `internal/core/coretest` (package `coretest`).
- Hand-written infrastructure you use but **must not edit**:
  - `internal/core/app.go`: `App`, `Config`, the runner interfaces, `KW`/`KV` keyword lists, `CommandError`.
  - `internal/core/changeset.go`: the Ecto.Changeset port.
  - `internal/core/repo.go`: the Ecto.Repo port (`Insert`, `Update`, `Delete`, `Get`, `All`, `One`, `MustOne`, `Scalar`, `Exec`, `From`, `Columns`, `SQ`).
  - `internal/core/repo_helpers.go`: `Pinchflat.Repo` (`InsertUniqueJob`, `MaybeLimit`).
  - `internal/core/preload.go`: association loading.
  - `internal/db`: `db.UTCDateTime`, `db.Date`, `db.JSON[T]`, `db.NestedStringArray`, `db.CompileRegex`, `db.EncodeJSON`.
  - `internal/obanlite`: jobs.
  - `internal/core/coretest/{app,mock,mocks}.go`: the test harness.

  If you need one of these to change, **stop and say so in your report** instead of editing it.

## Names

The package is flat, so each Elixir module gets a unique Go prefix. Use the prefix from the table at the bottom of this file.

| Elixir | Go |
|---|---|
| public `def foo_bar` in module with prefix `Sources` | `SourcesFooBar` |
| private `defp foo_bar` | `sourcesFooBar` (lower-case prefix) |
| `get_source!` and `get_source` both exist | `SourcesGetSourceBang` and `SourcesGetSource`; if only one exists, drop the `!` |
| predicate `pending_download?` | `...PendingDownload` (returns `bool`) |
| clauses that differ only by struct type, e.g. `build_output_path_for(%Source{})` / `(%MediaItem{})` | one function per type with a suffix: `DownloadOptionBuilderBuildOutputPathForSource`, `...ForMediaItem` |
| module attribute `@foo 123` | unexported `const`/`var` `sourcesFoo` |
| schema struct `%Source{}` | type `Source` |
| other structs (`%Pinchflat.YtDlp.Media{}`) | type named by the prefix: `YtDlpMedia` |

## Functions vs methods

- A function is a **method on `*App`** if it (or anything it calls) uses `Repo`, `Oban`, `Settings`, `Application.get_env`, a runner (yt-dlp, apprise, user scripts, HTTP), or the filesystem config directories. When unsure, make it a method.
- Otherwise (pure helpers: `utils/*`, the output path parser, formatting) it is a plain function.
- Every method and every function that does I/O takes `ctx context.Context` first.
- Use `a.Q(ctx)` wherever Elixir used `Repo`. Use `a.InTx(ctx, func(ctx context.Context) error {...})` for `Repo.transaction`.

## Types

- Integers → `int` (ids and foreign keys → `int64`). Floats → `float64`.
- Atoms → `string` (declare typed string constants for enumerated atoms).
- Keyword lists (including `opts \\ []`) → `core.KW`. Build them with `Flag("no_warnings")` for a bare atom and `Opt("output", path)` for `{:output, path}`; read them with `kw.Get`, `kw.GetOr`, `kw.Bool`, `kw.String`, `kw.HasFlag`, `kw.Contains`.
- Changeset attrs and loose maps with atom keys → `core.Attrs` (`map[string]any`, key = field name as a string).
- Maps with a fixed known shape that are returned to callers → a small named struct is fine, but keep field names equal to the Elixir keys (as json tags where relevant).
- `DateTime` in function params and returns → `time.Time` (UTC). `Date` → `time.Time` at midnight UTC.
- `nil`-able values → pointers (`*string`, `*int`, `*time.Time`). Use the `Ptr(v)` / `Deref(p)` helpers in `internal/core/ptr.go`.

### Return values

| Elixir | Go |
|---|---|
| `{:ok, x} \| {:error, reason}` | `(X, error)` |
| `:ok \| {:error, reason}` | `error` |
| `{:error, output, status}` from a command | `*core.CommandError` |
| raises (bang functions, `Repo.get!`) | return an `error` (`core.ErrNotFound` for missing records) |
| `{:ok, x}` only | `(X, error)` if it can fail for I/O reasons, else `X` |
| `{:error, %Ecto.Changeset{}}` | `*core.ChangesetError` (from `Insert`/`Update`); get the changeset with `core.AsChangesetError(err)` |

Keep `{:error, reason}` **reason strings identical**: tests and the UI compare them.

## Schemas

For each `schema "table"`:

```go
type Source struct {
    ID             int64          `db:"id"`
    CollectionName string         `db:"collection_name"`
    CollectionType CollectionType `db:"collection_type" enum:"channel,playlist"`
    CustomName     string         `db:"custom_name"`
    Description    *string        `db:"description"`
    DownloadCutoffDate *db.Date   `db:"download_cutoff_date"`
    LastIndexedAt  *db.UTCDateTime `db:"last_indexed_at"`
    ...
    InsertedAt db.UTCDateTime `db:"inserted_at"`
    UpdatedAt  db.UTCDateTime `db:"updated_at"`

    // Associations: never columns, loaded with preload helpers.
    MediaProfile *MediaProfile   `db:"-"`
    Metadata     *SourceMetadata `db:"-"`
    MediaItems   []*MediaItem    `db:"-"`
}

func (Source) TableName() string { return "sources" }
```

Column rules (check `testdata/elixir/golden/schema.sql` for NOT NULL and defaults):

- Every column in the table gets a field, **including columns the Ecto schema doesn't declare** (such as `sources.series_directory`), so `SELECT *`-style scans work. Mark those with a `// not in Ecto schema` comment.
- Nullable column → pointer type; NOT NULL column → value type.
- `:utc_datetime` → `db.UTCDateTime`; `:date` → `db.Date`.
- `:boolean` → `bool`; `:integer` → `int` (FK/ids `int64`); `:string` → `string`.
- `{:array, :string}` → `db.JSON[[]string]`; `{:array, {:array, :string}}` → `db.NestedStringArray`; `:map` → `db.JSON[map[string]any]`.
- `Ecto.Enum` → a named string type with constants, plus an `enum:"a,b,c"` tag listing the values.
- Ecto `default:` values → a `NewSource()`-style constructor returning the struct with defaults set. Insert paths start from it (Ecto starts from `%Source{}` which carries the defaults).
- Virtual fields → `db:"-"`, except where a query selects into them (then use the alias as the tag, e.g. `db:"matching_search_term"`, and have queries that don't select it use `Columns` explicitly).

Changesets become a function on the schema file:

```go
// Source.changeset(source, attrs, validation_stage)
func SourceChangeset(source *Source, attrs Attrs, validationStage string) *Changeset
```

These port line for line: `Cast`, `DynamicDefault`, `ValidateRequired`, `ValidateFormat` (pass the Elixir regex source as a string; it is compiled with PCRE-like semantics), `ValidateNumber`, `UniqueConstraint`, `CastAssoc`, `AddError`, `GetField`, `GetChange`, `PutChange`.

## Queries

Use squirrel through `core.SQ` (`?` placeholders) and the repo helpers:

```go
q := From[Source]("s").Join("media_profiles mp ON mp.id = s.media_profile_id").
    Where(sq.Eq{"s.enabled": true}).OrderBy("s.custom_name COLLATE NOCASE")
sources, err := All[Source](ctx, a.Q(ctx), q)
```

- Ecto `dynamic(...)` / `fragment(...)` → `sq.Expr("<the same SQL>", args...)`. **Copy fragment SQL verbatim.**
- Composable query modules (`MediaQuery`, `SourcesQuery`, ...) return `sq.Sqlizer` predicates, or functions `func(sq.SelectBuilder) sq.SelectBuilder`.
- `Repo.preload` → the helpers in `preload.go`.
- `Repo.update_all` / `delete_all` → `Exec(ctx, a.Q(ctx), SQ.Update(...)...)`.
- Timestamps as query args: pass `db.UTCDateTime{Time: t}` (or `db.Now()`), never a bare `time.Time`.
- Booleans in SQL compare as `= 1` / `= 0` or `= true` / `= false` (SQLite accepts both); keep whatever the Elixir fragment used.

### Query modules (already written, don't rewrite)

- `MediaQuery` (`media__media_query.go`): `MediaQueryNew()` returns a `*MediaQ` (`media_items AS mi`). Chain `.RequireAssoc("source" | "media_profile" | "media_items_search_index")`, `.Where(pred)`, `.MatchingSearchTerm(term)` and `.Map(func(sq.SelectBuilder) sq.SelectBuilder)`, then run it with `All[MediaItem](ctx, a.Q(ctx), q)`. Predicates such as `MediaQueryPending()` are `sq.Sqlizer` values; negate one with `Not(pred)`. Aliases match the Ecto binding names: `mi`, `source`, `media_profile`.
- `SourcesQueryNew()` (`sources AS s`), `ProfilesQueryNew()` (`media_profiles AS mp`) and `TasksQueryNew()` (`tasks AS t`) return `sq.SelectBuilder`. `TasksQueryJoinJob` joins `oban_jobs AS j`.
- Virtual fields that queries select into use `db:"name,virtual"` (e.g. `MediaItem.MatchingSearchTerm`).

### Real runners

The swappable modules are types that implement the `app.go` interfaces:
- `YtDlpCommandRunner{App}`
- `NotificationsCommandRunner{App}`
- `UserScriptsCommandRunner{App}` (plus `RunWithResult` for the full `{:ok, output, code}`)
- `HTTPClientImpl{}`

Everything else calls them through `a.YtDlp`, `a.Apprise`, `a.UserScripts` and `a.HTTP`, never directly, so tests can mock them.

### Boot tasks

The GenServers are gone. `(a *App) PreJobStartupTasksInit(ctx) error` and `PostBootStartupTasksInit(ctx) error` do the init work, and `main` calls them. `FileFollowerServer` is a struct with a goroutine: `FileFollowerServerStartLink(ctx, poll)`, `.WatchFile(path, handler)`, `.Stop()`.

### Tricky returns already decided

- `MediaDownloaderDownloadForMediaItem` returns `(*MediaDownloaderResult, error)`: `{:recovered, ...}` sets `Recovered`, and `{:error, atom, msg}` is a `*MediaDownloaderError{Reason, Message}`.
- `TasksCreateJobWithTask(ctx, obanlite.JobSpec, record)`: the job changeset (`Worker.new(args, opts)`) becomes a `JobSpec`.

## Jobs (Oban workers)

```go
const MediaDownloadWorkerName = "Pinchflat.Downloading.MediaDownloadWorker" // exact Elixir module name

var mediaDownloadWorkerOpts = obanlite.WorkerOpts{ // from `use Oban.Worker, ...`
    Queue: "media_fetching", Priority: 5,
    Unique: &obanlite.UniqueOpts{Period: obanlite.Infinity, States: []string{"available", "scheduled", "retryable", "executing"}},
    Tags:   []string{"media_item", "media_fetching", "show_in_dashboard"},
}

// perform/1
func (a *App) MediaDownloadWorkerPerform(ctx context.Context, job *obanlite.Job) error
```

- Return `nil` for `:ok`/`{:ok, _}`, an `error` for `{:error, _}`, `obanlite.Cancel(err)` for `{:cancel, _}`, and `obanlite.Snooze(n)` for `{:snooze, n}`.
- Decode args with `job.DecodeArgs(&struct{ ID int64 `json:"id"`; Force bool `json:"force"` }{})`. Keep the exact JSON key names, including `"quality_upgrade?"`.
- Insert with `a.Oban.Insert(ctx, a.Q(ctx), obanlite.JobSpec{Worker: ..., Args: map[string]any{...}, ScheduleIn: secs})`.
- `Repo.insert_unique_job` → `a.InsertUniqueJob(ctx, spec)`, which returns `(job, duplicate bool, err)`.
- Workers are registered centrally in `internal/core/workers.go`; don't register in your file.

## Processes, files, logging, JSON

- `System.cmd` goes through the ported `CliUtils` (W1). Never call `exec` directly elsewhere.
- `File.*` → `os.*`; `Path.join` → `filepath.Join`.
- `Logger.info/warning/error/debug` → `slog.Info/Warn/Error/Debug` with the same message text.
- `Jason.encode!` → `db.EncodeJSON`, which matches Jason: no HTML escaping, compact. `Jason.decode!` → `core.DecodeJSON` (`internal/core/json.go`), which uses `json.Number` for numbers.
- `DateTime.utc_now()` → `time.Now().UTC()`, or `db.Now()` when storing it.
- `Timex`/`Calendar.strftime` → Go `time` formatting; reproduce the exact output format.
- Elixir `~r//` regexes → `db.CompileRegex` (PCRE-like, via `regexp2`). Use Go's `regexp` only when the pattern has no lookarounds or backreferences.

## Tests

- One Go test function per `describe` block: `describe "perform/1"` in `media_download_worker_test.exs` → `func TestMediaDownloadWorker_Perform(t *testing.T)`. Tests outside any `describe` go in `TestMediaDownloadWorker`.
- Each Elixir `test "name"` → `t.Run("name", ...)` with **the identical string**. The parity script (`docs/go-port/parity.py`) matches these names exactly.
- Setup: `ta := coretest.NewApp(t)` gives `ta.App`, `ta.Ctx` and the mocks, replacing DataCase and the Ecto sandbox. Create a new `ta` inside each `t.Run`, because Elixir tests are isolated.
- Mox → the coretest mocks:
  - `expect(YtDlpRunnerMock, :run, fn url, action, opts, ot, addl -> ... end)` → `ta.YtDlpMock.Run.Expect(func(url, action string, opts core.KW, ot string, addl core.KW) (string, error) { ... })`. A 4-arity Elixir mock ignores `addl`.
  - `expect(M, :f, 2, fn)` → `ExpectN(2, fn)`; `stub` → `Stub`.
- `assert_enqueued(worker: W, args: %{"id" => 1})` → `ta.Oban.AssertEnqueued(t, obanlite.Match{Worker: core.WName, Args: map[string]any{"id": 1}})`; `refute_enqueued` → `RefuteEnqueued`; `perform_job(W, args)` → `ta.Oban.PerformJob(ctx, core.WName, args)`; `all_enqueued` → `ta.Oban.Enqueued(t, obanlite.Match{...})`.
- `errors_on(changeset)` → `cs.ErrorMap()`.
- Fixtures → `coretest.SourceFixture(t, ta, core.Attrs{...})` and the others. Helpers from `testing_helper_methods.ex` → `coretest.Now()`, `coretest.NowMinus(n, "day")`, `coretest.RenderMetadata(name)`, and so on.
- Assertions: plain `if got != want { t.Errorf(...) }`, or `reflect.DeepEqual` for structs and slices. No third-party assertion libraries.
- Files under `test/support/files` stay where they are; reference them with `coretest.RepoPath("test/support/files/...")`.
- `@tag :skip` tests → `t.Skip("skipped in Elixir too")`.

## Workflow for each manifest row

1. Read the Elixir source and its test file completely.
2. Replace the skeleton stubs in the target `.go` file with the real implementation. Add private helpers to that file only.
3. Port the test file.
4. Run `go build ./... && go vet ./internal/core/ && go test ./internal/core/ -run '^Test<Prefix>'`, and iterate until it passes.
5. Report: what you ported, the test count (Elixir vs Go), and any deviations or skeleton/infra problems. Don't edit files outside your row.

## Module prefix table

Generated. Where a prefix differs from the module's last segment, that's to avoid a collision.

| Elixir module | Go prefix | File |
|---|---|---|
| `Pinchflat.Application` | `Application` | `lib/pinchflat/application.ex` |
| `Pinchflat.Boot.PostBootStartupTasks` | `PostBootStartupTasks` | `lib/pinchflat/boot/post_boot_startup_tasks.ex` |
| `Pinchflat.Boot.PreJobStartupTasks` | `PreJobStartupTasks` | `lib/pinchflat/boot/pre_job_startup_tasks.ex` |
| `Pinchflat.Downloading.DownloadOptionBuilder` | `DownloadOptionBuilder` | `lib/pinchflat/downloading/download_option_builder.ex` |
| `Pinchflat.Downloading.DownloadingHelpers` | `DownloadingHelpers` | `lib/pinchflat/downloading/downloading_helpers.ex` |
| `Pinchflat.Downloading.MediaDownloadWorker` | `MediaDownloadWorker` | `lib/pinchflat/downloading/media_download_worker.ex` |
| `Pinchflat.Downloading.MediaDownloader` | `MediaDownloader` | `lib/pinchflat/downloading/media_downloader.ex` |
| `Pinchflat.Downloading.MediaQualityUpgradeWorker` | `MediaQualityUpgradeWorker` | `lib/pinchflat/downloading/media_quality_upgrade_worker.ex` |
| `Pinchflat.Downloading.MediaRetentionWorker` | `MediaRetentionWorker` | `lib/pinchflat/downloading/media_retention_worker.ex` |
| `Pinchflat.Downloading.OutputPath.Base` | `OutputPathBase` | `lib/pinchflat/downloading/output_path/base.ex` |
| `Pinchflat.Downloading.OutputPath.Parser` | `OutputPathParser` | `lib/pinchflat/downloading/output_path/parser.ex` |
| `Pinchflat.Downloading.OutputPathBuilder` | `OutputPathBuilder` | `lib/pinchflat/downloading/output_path_builder.ex` |
| `Pinchflat.Downloading.QualityOptionBuilder` | `QualityOptionBuilder` | `lib/pinchflat/downloading/quality_option_builder.ex` |
| `Pinchflat.FastIndexing.FastIndexingHelpers` | `FastIndexingHelpers` | `lib/pinchflat/fast_indexing/fast_indexing_helpers.ex` |
| `Pinchflat.FastIndexing.FastIndexingWorker` | `FastIndexingWorker` | `lib/pinchflat/fast_indexing/fast_indexing_worker.ex` |
| `Pinchflat.FastIndexing.YoutubeApi` | `YoutubeApi` | `lib/pinchflat/fast_indexing/youtube_api.ex` |
| `Pinchflat.FastIndexing.YoutubeBehaviour` | — (YoutubeBehaviour interface (declare in fast_indexing__youtube_api.go)) | `lib/pinchflat/fast_indexing/youtube_behaviour.ex` |
| `Pinchflat.FastIndexing.YoutubeRss` | `YoutubeRss` | `lib/pinchflat/fast_indexing/youtube_rss.ex` |
| `Pinchflat.HTTP.HTTPBehaviour` | — (HTTPClient interface (app.go)) | `lib/pinchflat/http/http_behaviour.ex` |
| `Pinchflat.HTTP.HTTPClient` | `HTTPClient` | `lib/pinchflat/http/http_client.ex` |
| `Pinchflat.Lifecycle.Notifications.AppriseCommandRunner` | — (AppriseRunner interface (app.go)) | `lib/pinchflat/lifecycle/notifications/apprise_command_runner.ex` |
| `Pinchflat.Lifecycle.Notifications.CommandRunner` | `NotificationsCommandRunner` | `lib/pinchflat/lifecycle/notifications/command_runner.ex` |
| `Pinchflat.Lifecycle.Notifications.SourceNotifications` | `SourceNotifications` | `lib/pinchflat/lifecycle/notifications/source_notifications.ex` |
| `Pinchflat.Lifecycle.UserScripts.CommandRunner` | `UserScriptsCommandRunner` | `lib/pinchflat/lifecycle/user_scripts/command_runner.ex` |
| `Pinchflat.Lifecycle.UserScripts.UserScriptCommandRunner` | — (UserScriptRunner interface (app.go)) | `lib/pinchflat/lifecycle/user_scripts/user_script_command_runner.ex` |
| `Pinchflat.Media.FileSyncing` | `FileSyncing` | `lib/pinchflat/media/file_syncing.ex` |
| `Pinchflat.Media.FileSyncingWorker` | `FileSyncingWorker` | `lib/pinchflat/media/file_syncing_worker.ex` |
| `Pinchflat.Media` | `Media` | `lib/pinchflat/media/media.ex` |
| `Pinchflat.Media.MediaItem` | `MediaItem` | `lib/pinchflat/media/media_item.ex` |
| `Pinchflat.Media.MediaItemsSearchIndex` | `MediaItemsSearchIndex` | `lib/pinchflat/media/media_items_search_index.ex` |
| `Pinchflat.Media.MediaQuery` | `MediaQuery` | `lib/pinchflat/media/media_query.ex` |
| `Pinchflat.Metadata.MediaMetadata` | `MediaMetadata` | `lib/pinchflat/metadata/media_metadata.ex` |
| `Pinchflat.Metadata.MetadataFileHelpers` | `MetadataFileHelpers` | `lib/pinchflat/metadata/metadata_file_helpers.ex` |
| `Pinchflat.Metadata.MetadataParser` | `MetadataParser` | `lib/pinchflat/metadata/metadata_parser.ex` |
| `Pinchflat.Metadata.NfoBuilder` | `NfoBuilder` | `lib/pinchflat/metadata/nfo_builder.ex` |
| `Pinchflat.Metadata.SourceImageParser` | `SourceImageParser` | `lib/pinchflat/metadata/source_image_parser.ex` |
| `Pinchflat.Metadata.SourceMetadata` | `SourceMetadata` | `lib/pinchflat/metadata/source_metadata.ex` |
| `Pinchflat.Metadata.SourceMetadataStorageWorker` | `SourceMetadataStorageWorker` | `lib/pinchflat/metadata/source_metadata_storage_worker.ex` |
| `Pinchflat.Podcasts.OpmlFeedBuilder` | `OpmlFeedBuilder` | `lib/pinchflat/podcasts/opml_feed_builder.ex` |
| `Pinchflat.Podcasts.PodcastHelpers` | `PodcastHelpers` | `lib/pinchflat/podcasts/podcast_helpers.ex` |
| `Pinchflat.Podcasts.RssFeedBuilder` | `RssFeedBuilder` | `lib/pinchflat/podcasts/rss_feed_builder.ex` |
| `Pinchflat.Profiles.MediaProfile` | `MediaProfile` | `lib/pinchflat/profiles/media_profile.ex` |
| `Pinchflat.Profiles.MediaProfileDeletionWorker` | `MediaProfileDeletionWorker` | `lib/pinchflat/profiles/media_profile_deletion_worker.ex` |
| `Pinchflat.Profiles` | `Profiles` | `lib/pinchflat/profiles/profiles.ex` |
| `Pinchflat.Profiles.ProfilesQuery` | `ProfilesQuery` | `lib/pinchflat/profiles/profiles_query.ex` |
| `Pinchflat.Settings.Setting` | `Setting` | `lib/pinchflat/settings/setting.ex` |
| `Pinchflat.Settings` | `Settings` | `lib/pinchflat/settings/settings.ex` |
| `Pinchflat.SlowIndexing.FileFollowerServer` | `FileFollowerServer` | `lib/pinchflat/slow_indexing/file_follower_server.ex` |
| `Pinchflat.SlowIndexing.MediaCollectionIndexingWorker` | `MediaCollectionIndexingWorker` | `lib/pinchflat/slow_indexing/media_collection_indexing_worker.ex` |
| `Pinchflat.SlowIndexing.SlowIndexingHelpers` | `SlowIndexingHelpers` | `lib/pinchflat/slow_indexing/slow_indexing_helpers.ex` |
| `Pinchflat.Sources.Source` | `Source` | `lib/pinchflat/sources/source.ex` |
| `Pinchflat.Sources.SourceDeletionWorker` | `SourceDeletionWorker` | `lib/pinchflat/sources/source_deletion_worker.ex` |
| `Pinchflat.Sources` | `Sources` | `lib/pinchflat/sources/sources.ex` |
| `Pinchflat.Sources.SourcesQuery` | `SourcesQuery` | `lib/pinchflat/sources/sources_query.ex` |
| `Pinchflat.Tasks.Task` | `Task` | `lib/pinchflat/tasks/task.ex` |
| `Pinchflat.Tasks` | `Tasks` | `lib/pinchflat/tasks/tasks.ex` |
| `Pinchflat.Tasks.TasksQuery` | `TasksQuery` | `lib/pinchflat/tasks/tasks_query.ex` |
| `Pinchflat.Utils.ChangesetUtils` | `ChangesetUtils` | `lib/pinchflat/utils/changeset_utils.ex` |
| `Pinchflat.Utils.CliUtils` | `CliUtils` | `lib/pinchflat/utils/cli_utils.ex` |
| `Pinchflat.Utils.FilesystemUtils` | `FilesystemUtils` | `lib/pinchflat/utils/filesystem_utils.ex` |
| `Pinchflat.Utils.FunctionUtils` | `FunctionUtils` | `lib/pinchflat/utils/function_utils.ex` |
| `Pinchflat.Utils.MapUtils` | `MapUtils` | `lib/pinchflat/utils/map_utils.ex` |
| `Pinchflat.Utils.NumberUtils` | `NumberUtils` | `lib/pinchflat/utils/number_utils.ex` |
| `Pinchflat.Utils.StringUtils` | `StringUtils` | `lib/pinchflat/utils/string_utils.ex` |
| `Pinchflat.Utils.XmlUtils` | `XmlUtils` | `lib/pinchflat/utils/xml_utils.ex` |
| `Pinchflat.YtDlp.CommandRunner` | `YtDlpCommandRunner` | `lib/pinchflat/yt_dlp/command_runner.ex` |
| `Pinchflat.YtDlp.Media` | `YtDlpMedia` | `lib/pinchflat/yt_dlp/media.ex` |
| `Pinchflat.YtDlp.MediaCollection` | `MediaCollection` | `lib/pinchflat/yt_dlp/media_collection.ex` |
| `Pinchflat.YtDlp.UpdateWorker` | `UpdateWorker` | `lib/pinchflat/yt_dlp/update_worker.ex` |
| `Pinchflat.YtDlp.YtDlpCommandRunner` | — (YtDlpRunner interface (app.go)) | `lib/pinchflat/yt_dlp/yt_dlp_command_runner.ex` |

Dropped (no Go file): `Pinchflat`, `Pinchflat.Mailer`, `Pinchflat.Release`, `Pinchflat.PromEx`, `Pinchflat.Boot.PostJobStartupTasks`. `Pinchflat.Repo` → `repo_helpers.go` (infrastructure).

