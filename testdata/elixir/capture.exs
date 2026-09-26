# Captures ground truth from the Elixir app for the Go port's compatibility tests.
#
# Run from the repo root (needs Elixir + sqlite3):
#
#     testdata/elixir/capture.sh
#
# Produces, in testdata/elixir/:
#   schema.db        - empty DB after all Ecto migrations
#   populated.db     - DB with realistic rows written by the Elixir code
#   golden/*         - outputs whose bytes the Go port must reproduce

import Ecto.Query
alias Pinchflat.Repo
alias Pinchflat.Settings
alias Pinchflat.Media.MediaItem
alias Pinchflat.Sources.Source
alias Pinchflat.ProfilesFixtures
alias Pinchflat.SourcesFixtures
alias Pinchflat.MediaFixtures
alias Pinchflat.Podcasts.RssFeedBuilder
alias Pinchflat.Podcasts.OpmlFeedBuilder

# Real runners (no executables configured means user scripts are a no-op)
Application.put_env(:pinchflat, :user_script_runner, Pinchflat.Lifecycle.UserScripts.CommandRunner)
Application.put_env(:pinchflat, :apprise_runner, Pinchflat.Lifecycle.Notifications.CommandRunner)
Faker.start()
:rand.seed(:exsss, {1, 2, 3})

golden = Path.join(["testdata", "elixir", "golden"])
File.mkdir_p!(golden)

# --- Profiles ---
p_default = ProfilesFixtures.media_profile_fixture(%{name: "Default"})

p_audio =
  ProfilesFixtures.media_profile_fixture(%{
    name: "Audio only",
    preferred_resolution: :audio,
    sponsorblock_behaviour: :remove,
    sponsorblock_categories: ["sponsor", "intro"],
    shorts_behaviour: :exclude,
    livestream_behaviour: :only,
    redownload_delay_days: 4,
    media_container: "mp4",
    audio_track: "en"
  })

p_marked = ProfilesFixtures.media_profile_fixture(%{name: "Marked", shorts_behaviour: :only})
{1, _} = Repo.update_all(from(p in Pinchflat.Profiles.MediaProfile, where: p.id == ^p_marked.id), set: [marked_for_deletion_at: DateTime.utc_now() |> DateTime.truncate(:second)])

# --- Sources ---
s_channel =
  SourcesFixtures.source_fixture(%{
    media_profile_id: p_default.id,
    custom_name: "Channel Source",
    collection_type: "channel",
    title_filter_regex: "(?i)^(?!.*trailer).*$",
    download_cutoff_date: ~D[2024-01-15],
    retention_period_days: 30,
    cookie_behaviour: :when_needed,
    min_duration_seconds: 60,
    max_duration_seconds: 3600,
    fast_index: true
  })

s_playlist =
  SourcesFixtures.source_fixture(%{
    media_profile_id: p_audio.id,
    custom_name: "Playlist Source",
    collection_type: "playlist",
    original_url: "https://www.youtube.com/playlist?list=PLabc123",
    cookie_behaviour: :all_operations,
    output_path_template_override: "/{{ source_custom_name }}/{{ title }}.{{ ext }}"
  })

s_meta = SourcesFixtures.source_with_metadata(%{media_profile_id: p_default.id, custom_name: "With Metadata", enabled: false})

# --- Media items ---
now = DateTime.utc_now() |> DateTime.truncate(:second)

mi_downloaded =
  MediaFixtures.media_item_fixture(%{
    source_id: s_channel.id,
    title: "Downloaded video — ünïcödé & <xml> \"quotes\"",
    description: "Line one\nLine two",
    media_downloaded_at: now,
    media_size_bytes: 123_456_789,
    duration_seconds: 754,
    subtitle_filepaths: [["en", "/video/a.en.srt"], ["de", "/video/a.de.srt"]],
    thumbnail_filepath: "/video/a.jpg",
    metadata_filepath: "/metadata/a.json.gz",
    nfo_filepath: "/video/a.nfo",
    uploaded_at: ~U[2024-02-03 04:05:06Z],
    playlist_index: 3
  })

mi_pending = MediaFixtures.media_item_fixture(%{source_id: s_channel.id, media_filepath: nil, title: "Pending video", uploaded_at: ~U[2024-05-06 07:08:09Z]})
mi_short = MediaFixtures.media_item_fixture(%{source_id: s_playlist.id, short_form_content: true, livestream: true, prevent_download: true, prevent_culling: true})

mi_culled =
  MediaFixtures.media_item_fixture(%{
    source_id: s_playlist.id,
    media_filepath: nil,
    culled_at: now,
    media_redownloaded_at: now,
    last_error: "ERROR: [youtube] abc: Video unavailable",
    predicted_media_filepath: "/downloads/predicted.mkv"
  })

_mi_meta = MediaFixtures.media_item_with_metadata(%{source_id: s_meta.id})

# Legacy uploaded_at values produced by migration 20240528180212 and the column default
mi_legacy = MediaFixtures.media_item_fixture(%{source_id: s_channel.id, title: "Legacy uploaded_at"})
Repo.query!("UPDATE media_items SET uploaded_at = '2023-11-12T00:00:00' WHERE id = ?", [mi_legacy.id])
Repo.query!(
  "INSERT INTO media_items (media_id, title, original_url, source_id, inserted_at, updated_at) VALUES ('legacydefault', 'Column default uploaded_at', 'https://www.youtube.com/watch?v=legacydefault', ?, ?, ?)",
  [s_channel.id, DateTime.to_iso8601(now), DateTime.to_iso8601(now)]
)

# --- Settings ---
Settings.set(youtube_api_key: "key-one,key-two")
Settings.set(apprise_server: "json://localhost:8080 discord://a/b")
Settings.set(download_throughput_limit: "4.2M")
Settings.set(extractor_sleep_interval_seconds: 5)
Settings.set(restrict_filenames: true)
Settings.set(onboarding: false)

# --- Jobs + tasks, via the real kickoff functions ---
{:ok, _} = Pinchflat.Downloading.MediaDownloadWorker.kickoff_with_task(mi_pending)
{:ok, _} = Pinchflat.Downloading.MediaDownloadWorker.kickoff_with_task(mi_short, %{force: true})
{:ok, _} = Pinchflat.Downloading.MediaDownloadWorker.kickoff_with_task(mi_culled, %{quality_upgrade?: true})
{:ok, _} = Pinchflat.SlowIndexing.MediaCollectionIndexingWorker.kickoff_with_task(s_channel, %{}, schedule_in: 3600)
{:ok, _} = Pinchflat.SlowIndexing.MediaCollectionIndexingWorker.kickoff_with_task(s_playlist, %{force: true})
{:ok, _} = Pinchflat.FastIndexing.FastIndexingWorker.kickoff_with_task(s_channel)
{:ok, _} = Pinchflat.Metadata.SourceMetadataStorageWorker.kickoff_with_task(s_meta)
{:ok, _} = Pinchflat.Media.FileSyncingWorker.kickoff_with_task(s_channel)
{:ok, _} = Pinchflat.YtDlp.UpdateWorker.kickoff()
{:ok, _} = Pinchflat.Sources.SourceDeletionWorker.kickoff(s_meta, %{delete_files: true}, schedule_in: 86_400)
{:ok, _} = Pinchflat.Profiles.MediaProfileDeletionWorker.kickoff(p_marked, %{delete_files: false}, schedule_in: 86_400)
{:ok, _} = Oban.insert(Pinchflat.Downloading.MediaRetentionWorker.new(%{}))
{:ok, _} = Oban.insert(Pinchflat.Downloading.MediaQualityUpgradeWorker.new(%{}))

# A duplicate unique insert, to record what Oban does on conflict
dup = Pinchflat.Downloading.MediaDownloadWorker.new(%{id: mi_pending.id})
{:duplicate, _} = Repo.insert_unique_job(dup)

# Move a few jobs into every other state the way Oban writes them
ts = fn -> DateTime.utc_now() |> DateTime.to_iso8601() end
err = fn attempt -> Jason.encode!([%{at: ts.(), attempt: attempt, error: "** (RuntimeError) boom"}]) end

states = [
  {"Pinchflat.YtDlp.UpdateWorker", "completed", "completed_at"},
  {"Pinchflat.Downloading.MediaRetentionWorker", "discarded", "discarded_at"},
  {"Pinchflat.Downloading.MediaQualityUpgradeWorker", "cancelled", "cancelled_at"},
  {"Pinchflat.Media.FileSyncingWorker", "executing", nil},
  {"Pinchflat.Metadata.SourceMetadataStorageWorker", "retryable", nil}
]

for {worker, state, col} <- states do
  extra = if col, do: ", #{col} = ?4", else: ""
  args = [state, 1, err.(1), ts.(), worker]
  args = if col, do: List.insert_at(args, 3, ts.()), else: args

  sql =
    "UPDATE oban_jobs SET state = ?1, attempt = ?2, errors = ?3#{extra}, attempted_at = ?#{length(args) - 1}, attempted_by = '[\"capture\"]' WHERE worker = ?#{length(args)}"

  Repo.query!(sql, args)
end

# --- Golden outputs ---
reload = fn
  %MediaItem{} = m -> Repo.preload(Repo.reload!(m), [:metadata, source: :media_profile], force: true)
  %Source{} = s -> Repo.preload(Repo.reload!(s), [:media_profile, :metadata], force: true)
end

File.write!(Path.join(golden, "user_script_media_item.json"), Jason.encode!(reload.(mi_downloaded), pretty: true))
File.write!(Path.join(golden, "user_script_source.json"), Jason.encode!(reload.(s_channel), pretty: true))
File.write!(Path.join(golden, "user_script_media_profile.json"), Jason.encode!(Repo.reload!(p_audio), pretty: true))

for {name, source} <- [{"channel", s_channel}, {"playlist", s_playlist}] do
  xml = RssFeedBuilder.build(reload.(source), url_base: "http://pinchflat.test:8945")
  File.write!(Path.join(golden, "rss_#{name}.xml"), xml)
end

File.write!(
  Path.join(golden, "opml.xml"),
  OpmlFeedBuilder.build("http://pinchflat.test:8945", Pinchflat.Podcasts.PodcastHelpers.opml_sources())
)

IO.puts("capture complete")
