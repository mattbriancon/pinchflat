# Phase 2: idiomatic Go

Phase 1 ported Pinchflat file by file, keeping Elixir's shapes. It used:
- one `internal/core` package with `ModulePrefix` names;
- keyword lists (`KW`/`Opt`/`Flag`) and `Attrs` maps;
- an Ecto changeset shim;
- Mox-style mocks;
- one Go test per Elixir test.

Phase 2 makes it ordinary Go and deletes whatever the Go version doesn't need.

**Rules:**
- Behaviour and the database stay exactly as they are. The compatibility suite (`internal/db`, `*compat*` tests, `testdata/elixir`) must stay green.
- Each stage is one or more commits on `claude/idiomatic-go`, verified with `go vet` and `go test ./...`.
- Each stage is followed by a `/simplify` pass, whose fixes are committed separately.

## Stages

1. **Dead code** (done). Remove what `deadcode ./cmd/pinchflat` reports and the tests that only covered it.
2. **Package split.** Break up `internal/core` along its real dependencies:
   - `internal/store`: entity types (Source, MediaItem, MediaProfile, Task, Setting, metadata rows), their queries and CRUD, plus settings. It depends only on `db`.
   - `internal/ytdlp`: the yt-dlp runner, option formatting and the `Media` parsing.
   - `internal/fsutil`: filesystem and command helpers (the old `utils__*`).
   - `internal/app`: the workflows (indexing, downloading, metadata, lifecycle scripts, podcasts, boot, workers) on top of those.
   Names become Go names: `SourcesGetSource` → `store.GetSource`, and `FilesystemUtilsExistsAndNonempty` → `fsutil.ExistsAndNonEmpty`.
3. **Typed data instead of maps.**
   - Replace the changeset shim and `Attrs` with typed input structs and `Validate() map[string][]string`.
   - Web forms bind to those structs.
   - Replace `KW` option lists with small option structs, or plain parameters.
4. **Tests.**
   - Collapse the one-test-per-Elixir-test files into table-driven tests.
   - Replace the Mox-style mocks with small fakes.
   - Add `t.Parallel()` where the per-test database copy allows it.
5. **Web.**
   - Simplify `internal/web` along the same lines.
   - Consider replacing Alpine with `<details>`, `<dialog>` and a little plain JS.
