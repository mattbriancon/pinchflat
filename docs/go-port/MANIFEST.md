# File-by-file port manifest

Generated from `git ls-files lib`. One row is one Haiku task: port the source file and its test file together.
`internal/core` and `internal/web` are single flat packages during the port (see STRATEGY.md §4.1), so the target file name encodes the old path with `__` in place of `/` (mechanical, collision-free, greppable).

| Wave | Elixir source | Go target | Elixir test to port | Notes |
|---|---|---|---|---|
| W0 | `lib/pinchflat/repo.ex` | `internal/core/repo.go` | `test/pinchflat/repo_test.exs` | helpers fold into internal/db + internal/obanlite |
| W1 | `lib/pinchflat/downloading/download_option_builder.ex` | `internal/core/downloading__download_option_builder.go` | `test/pinchflat/downloading/download_option_builder_test.exs` |  |
| W1 | `lib/pinchflat/downloading/output_path/base.ex` | `internal/core/downloading__output_path__base.go` | — |  |
| W1 | `lib/pinchflat/downloading/output_path/parser.ex` | `internal/core/downloading__output_path__parser.go` | `test/pinchflat/downloading/output_path/parser_test.exs` | nimble_parsec grammar -> hand-written parser |
| W1 | `lib/pinchflat/downloading/output_path_builder.ex` | `internal/core/downloading__output_path_builder.go` | `test/pinchflat/downloading/output_path_builder_test.exs` |  |
| W1 | `lib/pinchflat/downloading/quality_option_builder.ex` | `internal/core/downloading__quality_option_builder.go` | `test/pinchflat/downloading/quality_option_builder_test.exs` |  |
| W1 | `lib/pinchflat/fast_indexing/youtube_api.ex` | `internal/core/fast_indexing__youtube_api.go` | `test/pinchflat/fast_indexing/youtube_api_test.exs` |  |
| W1 | `lib/pinchflat/fast_indexing/youtube_rss.ex` | `internal/core/fast_indexing__youtube_rss.go` | `test/pinchflat/fast_indexing/youtube_rss_test.exs` |  |
| W1 | `lib/pinchflat/http/http_client.ex` | `internal/core/http__http_client.go` | — |  |
| W1 | `lib/pinchflat/lifecycle/notifications/command_runner.ex` | `internal/core/lifecycle__notifications__command_runner.go` | `test/pinchflat/lifecycle/notifications/command_runner_test.exs` |  |
| W1 | `lib/pinchflat/lifecycle/user_scripts/command_runner.ex` | `internal/core/lifecycle__user_scripts__command_runner.go` | `test/pinchflat/lifecycle/user_scripts/command_runner_test.exs` |  |
| W1 | `lib/pinchflat/metadata/metadata_file_helpers.ex` | `internal/core/metadata__metadata_file_helpers.go` | `test/pinchflat/metadata/metadata_file_helpers_test.exs` |  |
| W1 | `lib/pinchflat/metadata/metadata_parser.ex` | `internal/core/metadata__metadata_parser.go` | `test/pinchflat/metadata/metadata_parser_test.exs` |  |
| W1 | `lib/pinchflat/metadata/nfo_builder.ex` | `internal/core/metadata__nfo_builder.go` | `test/pinchflat/metadata/nfo_builder_test.exs` |  |
| W1 | `lib/pinchflat/metadata/source_image_parser.ex` | `internal/core/metadata__source_image_parser.go` | `test/pinchflat/metadata/source_image_parser_test.exs` |  |
| W1 | `lib/pinchflat/podcasts/opml_feed_builder.ex` | `internal/core/podcasts__opml_feed_builder.go` | `test/pinchflat/podcasts/opml_feed_builder_test.exs` |  |
| W1 | `lib/pinchflat/podcasts/podcast_helpers.ex` | `internal/core/podcasts__podcast_helpers.go` | `test/pinchflat/podcasts/podcast_helpers_test.exs` |  |
| W1 | `lib/pinchflat/podcasts/rss_feed_builder.ex` | `internal/core/podcasts__rss_feed_builder.go` | `test/pinchflat/podcasts/rss_feed_builder_test.exs` |  |
| W1 | `lib/pinchflat/slow_indexing/file_follower_server.ex` | `internal/core/slow_indexing__file_follower_server.go` | `test/pinchflat/slow_indexing/file_follower_server_test.exs` | GenServer -> goroutine tailing a file |
| W1 | `lib/pinchflat/utils/changeset_utils.ex` | `internal/core/utils__changeset_utils.go` | `test/pinchflat/utils/changeset_utils_test.exs` |  |
| W1 | `lib/pinchflat/utils/cli_utils.ex` | `internal/core/utils__cli_utils.go` | `test/pinchflat/utils/cli_utils_test.exs` |  |
| W1 | `lib/pinchflat/utils/filesystem_utils.ex` | `internal/core/utils__filesystem_utils.go` | `test/pinchflat/utils/filesystem_utils_test.exs` |  |
| W1 | `lib/pinchflat/utils/function_utils.ex` | `internal/core/utils__function_utils.go` | `test/pinchflat/utils/function_utils_test.exs` |  |
| W1 | `lib/pinchflat/utils/map_utils.ex` | `internal/core/utils__map_utils.go` | `test/pinchflat/utils/map_utils_test.exs` |  |
| W1 | `lib/pinchflat/utils/number_utils.ex` | `internal/core/utils__number_utils.go` | `test/pinchflat/utils/number_utils_test.exs` |  |
| W1 | `lib/pinchflat/utils/string_utils.ex` | `internal/core/utils__string_utils.go` | `test/pinchflat/utils/string_utils_test.exs` |  |
| W1 | `lib/pinchflat/utils/xml_utils.ex` | `internal/core/utils__xml_utils.go` | `test/pinchflat/utils/xml_utils_test.exs` |  |
| W1 | `lib/pinchflat/yt_dlp/command_runner.ex` | `internal/core/yt_dlp__command_runner.go` | `test/pinchflat/yt_dlp/command_runner_test.exs` |  |
| W1 | `lib/pinchflat/yt_dlp/media.ex` | `internal/core/yt_dlp__media.go` | `test/pinchflat/yt_dlp/media_test.exs` |  |
| W1 | `lib/pinchflat/yt_dlp/media_collection.ex` | `internal/core/yt_dlp__media_collection.go` | `test/pinchflat/yt_dlp/media_collection_test.exs` |  |
| W2 | `lib/pinchflat/media/media.ex` | `internal/core/media__media.go` | `test/pinchflat/media_test.exs` |  |
| W2 | `lib/pinchflat/media/media_item.ex` | `internal/core/media__media_item.go` | — | custom Jason.Encoder = user-script JSON contract |
| W2 | `lib/pinchflat/media/media_items_search_index.ex` | `internal/core/media__media_items_search_index.go` | — |  |
| W2 | `lib/pinchflat/media/media_query.ex` | `internal/core/media__media_query.go` | — | regexp_like/FTS5/date() fragments; squirrel composition |
| W2 | `lib/pinchflat/metadata/media_metadata.ex` | `internal/core/metadata__media_metadata.go` | — |  |
| W2 | `lib/pinchflat/metadata/source_metadata.ex` | `internal/core/metadata__source_metadata.go` | — |  |
| W2 | `lib/pinchflat/profiles/media_profile.ex` | `internal/core/profiles__media_profile.go` | — | custom Jason.Encoder = user-script JSON contract |
| W2 | `lib/pinchflat/profiles/profiles.ex` | `internal/core/profiles__profiles.go` | `test/pinchflat/profiles_test.exs` |  |
| W2 | `lib/pinchflat/profiles/profiles_query.ex` | `internal/core/profiles__profiles_query.go` | — |  |
| W2 | `lib/pinchflat/settings/setting.ex` | `internal/core/settings__setting.go` | — |  |
| W2 | `lib/pinchflat/settings/settings.ex` | `internal/core/settings__settings.go` | `test/pinchflat/settings_test.exs` |  |
| W2 | `lib/pinchflat/sources/source.ex` | `internal/core/sources__source.go` | — | custom Jason.Encoder = user-script JSON contract |
| W2 | `lib/pinchflat/sources/sources.ex` | `internal/core/sources__sources.go` | `test/pinchflat/sources_test.exs` |  |
| W2 | `lib/pinchflat/sources/sources_query.ex` | `internal/core/sources__sources_query.go` | — |  |
| W2 | `lib/pinchflat/tasks/task.ex` | `internal/core/tasks__task.go` | — |  |
| W2 | `lib/pinchflat/tasks/tasks.ex` | `internal/core/tasks__tasks.go` | `test/pinchflat/tasks_test.exs` |  |
| W2 | `lib/pinchflat/tasks/tasks_query.ex` | `internal/core/tasks__tasks_query.go` | — |  |
| W3 | `lib/pinchflat/application.ex` | `cmd/pinchflat/main.go` | — |  |
| W3 | `lib/pinchflat/boot/post_boot_startup_tasks.ex` | `internal/core/boot__post_boot_startup_tasks.go` | `test/pinchflat/boot/post_boot_startup_tasks_test.exs` |  |
| W3 | `lib/pinchflat/boot/pre_job_startup_tasks.ex` | `internal/core/boot__pre_job_startup_tasks.go` | `test/pinchflat/boot/pre_job_startup_tasks_test.exs` |  |
| W3 | `lib/pinchflat/downloading/downloading_helpers.ex` | `internal/core/downloading__downloading_helpers.go` | `test/pinchflat/downloading/downloading_helpers_test.exs` |  |
| W3 | `lib/pinchflat/downloading/media_download_worker.ex` | `internal/core/downloading__media_download_worker.go` | `test/pinchflat/downloading/media_download_worker_test.exs` |  |
| W3 | `lib/pinchflat/downloading/media_downloader.ex` | `internal/core/downloading__media_downloader.go` | `test/pinchflat/downloading/media_downloader_test.exs` |  |
| W3 | `lib/pinchflat/downloading/media_quality_upgrade_worker.ex` | `internal/core/downloading__media_quality_upgrade_worker.go` | `test/pinchflat/downloading/media_quality_upgrade_worker_test.exs` |  |
| W3 | `lib/pinchflat/downloading/media_retention_worker.ex` | `internal/core/downloading__media_retention_worker.go` | `test/pinchflat/downloading/media_retention_worker_test.exs` |  |
| W3 | `lib/pinchflat/fast_indexing/fast_indexing_helpers.ex` | `internal/core/fast_indexing__fast_indexing_helpers.go` | `test/pinchflat/fast_indexing/fast_indexing_helpers_test.exs` |  |
| W3 | `lib/pinchflat/fast_indexing/fast_indexing_worker.ex` | `internal/core/fast_indexing__fast_indexing_worker.go` | `test/pinchflat/fast_indexing/fast_indexing_worker_test.exs` |  |
| W3 | `lib/pinchflat/lifecycle/notifications/source_notifications.ex` | `internal/core/lifecycle__notifications__source_notifications.go` | `test/pinchflat/lifecycle/notifications/source_notifications_test.exs` |  |
| W3 | `lib/pinchflat/media/file_syncing.ex` | `internal/core/media__file_syncing.go` | `test/pinchflat/media/file_syncing_test.exs` |  |
| W3 | `lib/pinchflat/media/file_syncing_worker.ex` | `internal/core/media__file_syncing_worker.go` | `test/pinchflat/media/file_syncing_worker_test.exs` |  |
| W3 | `lib/pinchflat/metadata/source_metadata_storage_worker.ex` | `internal/core/metadata__source_metadata_storage_worker.go` | `test/pinchflat/metadata/source_metadata_storage_worker_test.exs` |  |
| W3 | `lib/pinchflat/profiles/media_profile_deletion_worker.ex` | `internal/core/profiles__media_profile_deletion_worker.go` | `test/pinchflat/profiles/media_profile_deletion_worker_test.exs` |  |
| W3 | `lib/pinchflat/slow_indexing/media_collection_indexing_worker.ex` | `internal/core/slow_indexing__media_collection_indexing_worker.go` | `test/pinchflat/slow_indexing/media_collection_indexing_worker_test.exs` |  |
| W3 | `lib/pinchflat/slow_indexing/slow_indexing_helpers.ex` | `internal/core/slow_indexing__slow_indexing_helpers.go` | `test/pinchflat/slow_indexing/slow_indexing_helpers_test.exs` |  |
| W3 | `lib/pinchflat/sources/source_deletion_worker.ex` | `internal/core/sources__source_deletion_worker.go` | `test/pinchflat/sources/source_deletion_worker_test.exs` |  |
| W3 | `lib/pinchflat/yt_dlp/update_worker.ex` | `internal/core/yt_dlp__update_worker.go` | `test/pinchflat/yt_dlp/update_worker_test.exs` |  |
| W4 | `lib/pinchflat_web/components/core_components.ex` | `internal/web/core_components.go` | — |  |
| W4 | `lib/pinchflat_web/components/custom_components/button_components.ex` | `internal/web/custom_components__button_components.go` | — |  |
| W4 | `lib/pinchflat_web/components/custom_components/tab_components.ex` | `internal/web/custom_components__tab_components.go` | — |  |
| W4 | `lib/pinchflat_web/components/custom_components/table_components.ex` | `internal/web/custom_components__table_components.go` | — |  |
| W4 | `lib/pinchflat_web/components/custom_components/text_components.ex` | `internal/web/custom_components__text_components.go` | — |  |
| W4 | `lib/pinchflat_web/components/layouts.ex` | `internal/web/layouts.go` | — |  |
| W4 | `lib/pinchflat_web/components/layouts/app.html.heex` | `internal/web/layouts__app.templ` | — |  |
| W4 | `lib/pinchflat_web/components/layouts/onboarding.html.heex` | `internal/web/layouts__onboarding.templ` | — |  |
| W4 | `lib/pinchflat_web/components/layouts/partials/donate_modal.heex` | `internal/web/layouts__partials__donate_modal.templ` | — |  |
| W4 | `lib/pinchflat_web/components/layouts/partials/footer.html.heex` | `internal/web/layouts__partials__footer.templ` | — |  |
| W4 | `lib/pinchflat_web/components/layouts/partials/header.html.heex` | `internal/web/layouts__partials__header.templ` | — |  |
| W4 | `lib/pinchflat_web/components/layouts/partials/upgrade_button_live.ex` | `internal/web/layouts__partials__upgrade_button_live.go` | — | LiveView -> handler + htmx |
| W4 | `lib/pinchflat_web/components/layouts/partials/upgrade_modal.heex` | `internal/web/layouts__partials__upgrade_modal.templ` | — |  |
| W4 | `lib/pinchflat_web/components/layouts/root.html.heex` | `internal/web/layouts__root.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/error_html.ex` | `internal/web/error_html.go` | `test/pinchflat_web/controllers/error_html_test.exs` |  |
| W4 | `lib/pinchflat_web/controllers/error_html/404.html.heex` | `internal/web/error_html__404.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/error_html/500.html.heex` | `internal/web/error_html__500.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/health_controller.ex` | `internal/web/health_controller.go` | `test/pinchflat_web/controllers/health_controller_test.exs` |  |
| W4 | `lib/pinchflat_web/controllers/media_items/media_item_controller.ex` | `internal/web/media_items__media_item_controller.go` | `test/pinchflat_web/controllers/media_item_controller_test.exs` |  |
| W4 | `lib/pinchflat_web/controllers/media_items/media_item_html.ex` | `internal/web/media_items__media_item_html.go` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_items/media_item_html/actions_dropdown.html.heex` | `internal/web/media_items__media_item_html__actions_dropdown.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_items/media_item_html/edit.html.heex` | `internal/web/media_items__media_item_html__edit.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_items/media_item_html/media_item_form.html.heex` | `internal/web/media_items__media_item_html__media_item_form.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_items/media_item_html/media_preview.heex` | `internal/web/media_items__media_item_html__media_preview.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_items/media_item_html/show.html.heex` | `internal/web/media_items__media_item_html__show.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_profiles/media_profile_controller.ex` | `internal/web/media_profiles__media_profile_controller.go` | `test/pinchflat_web/controllers/media_profile_controller_test.exs` |  |
| W4 | `lib/pinchflat_web/controllers/media_profiles/media_profile_html.ex` | `internal/web/media_profiles__media_profile_html.go` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_profiles/media_profile_html/actions_dropdown.html.heex` | `internal/web/media_profiles__media_profile_html__actions_dropdown.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_profiles/media_profile_html/edit.html.heex` | `internal/web/media_profiles__media_profile_html__edit.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_profiles/media_profile_html/index.html.heex` | `internal/web/media_profiles__media_profile_html__index.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_profiles/media_profile_html/media_profile_form.html.heex` | `internal/web/media_profiles__media_profile_html__media_profile_form.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_profiles/media_profile_html/new.html.heex` | `internal/web/media_profiles__media_profile_html__new.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_profiles/media_profile_html/output_template_help.html.heex` | `internal/web/media_profiles__media_profile_html__output_template_help.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/media_profiles/media_profile_html/show.html.heex` | `internal/web/media_profiles__media_profile_html__show.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/pages/page_controller.ex` | `internal/web/pages__page_controller.go` | `test/pinchflat_web/controllers/page_controller_test.exs` |  |
| W4 | `lib/pinchflat_web/controllers/pages/page_html.ex` | `internal/web/pages__page_html.go` | — |  |
| W4 | `lib/pinchflat_web/controllers/pages/page_html/history_table_live.ex` | `internal/web/pages__page_html__history_table_live.go` | — | LiveView -> handler + htmx + SSE (needs push) |
| W4 | `lib/pinchflat_web/controllers/pages/page_html/home.html.heex` | `internal/web/pages__page_html__home.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/pages/page_html/job_table_live.ex` | `internal/web/pages__page_html__job_table_live.go` | `test/pinchflat_web/controllers/pages/job_table_live_test.exs` | LiveView -> handler + htmx + SSE (needs push) |
| W4 | `lib/pinchflat_web/controllers/pages/page_html/onboarding_checklist.html.heex` | `internal/web/pages__page_html__onboarding_checklist.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/podcasts/podcast_controller.ex` | `internal/web/podcasts__podcast_controller.go` | `test/pinchflat_web/controllers/podcast_controller_test.exs` |  |
| W4 | `lib/pinchflat_web/controllers/searches/search_controller.ex` | `internal/web/searches__search_controller.go` | `test/pinchflat_web/controllers/search_controller_test.exs` |  |
| W4 | `lib/pinchflat_web/controllers/searches/search_html.ex` | `internal/web/searches__search_html.go` | — |  |
| W4 | `lib/pinchflat_web/controllers/searches/search_html/show.html.heex` | `internal/web/searches__search_html__show.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/settings/setting_controller.ex` | `internal/web/settings__setting_controller.go` | `test/pinchflat_web/controllers/setting_controller_test.exs` |  |
| W4 | `lib/pinchflat_web/controllers/settings/setting_html.ex` | `internal/web/settings__setting_html.go` | — |  |
| W4 | `lib/pinchflat_web/controllers/settings/setting_html/app_info.html.heex` | `internal/web/settings__setting_html__app_info.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/settings/setting_html/apprise_server_live.ex` | `internal/web/settings__setting_html__apprise_server_live.go` | `test/pinchflat_web/controllers/settings/apprise_server_live_test.exs` | LiveView -> handler + htmx |
| W4 | `lib/pinchflat_web/controllers/settings/setting_html/codec_settings_help.html.heex` | `internal/web/settings__setting_html__codec_settings_help.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/settings/setting_html/setting_form.html.heex` | `internal/web/settings__setting_html__setting_form.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/settings/setting_html/show.html.heex` | `internal/web/settings__setting_html__show.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_controller.ex` | `internal/web/sources__source_controller.go` | `test/pinchflat_web/controllers/source_controller_test.exs` |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_html.ex` | `internal/web/sources__source_html.go` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_html/actions_dropdown.html.heex` | `internal/web/sources__source_html__actions_dropdown.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_html/edit.html.heex` | `internal/web/sources__source_html__edit.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_html/fast_indexing_help.html.heex` | `internal/web/sources__source_html__fast_indexing_help.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_html/index.html.heex` | `internal/web/sources__source_html__index.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_html/media_item_table_live.ex` | `internal/web/sources__source_html__media_item_table_live.go` | `test/pinchflat_web/controllers/sources/media_item_table_live_test.exs` | LiveView -> handler + htmx |
| W4 | `lib/pinchflat_web/controllers/sources/source_html/new.html.heex` | `internal/web/sources__source_html__new.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_html/show.html.heex` | `internal/web/sources__source_html__show.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_html/source_form.html.heex` | `internal/web/sources__source_html__source_form.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_live/index_table_live.ex` | `internal/web/sources__source_live__index_table_live.go` | `test/pinchflat_web/controllers/sources/index_table_live_test.exs` | LiveView -> handler + htmx |
| W4 | `lib/pinchflat_web/controllers/sources/source_live/index_table_live.html.heex` | `internal/web/sources__source_live__index_table_live.templ` | — |  |
| W4 | `lib/pinchflat_web/controllers/sources/source_live/source_enable_toggle.ex` | `internal/web/sources__source_live__source_enable_toggle.go` | `test/pinchflat_web/controllers/sources/source_enable_toggle_test.exs` |  |
| W4 | `lib/pinchflat_web/endpoint.ex` | `internal/web/endpoint.go` | — | keep strip_trailing_extension + x-forwarded-proto base URL |
| W4 | `lib/pinchflat_web/helpers/pagination_helpers.ex` | `internal/web/helpers__pagination_helpers.go` | `test/pinchflat_web/helpers/pagination_helpers_test.exs` |  |
| W4 | `lib/pinchflat_web/helpers/sorting_helpers.ex` | `internal/web/helpers__sorting_helpers.go` | `test/pinchflat_web/helpers/sorting_helpers_test.exs` |  |
| W4 | `lib/pinchflat_web/plugs.ex` | `internal/web/plugs.go` | `test/pinchflat_web/plugs_test.exs` |  |
| W4 | `lib/pinchflat_web/router.ex` | `internal/web/router.go` | — | chi router; URLs must not change |
| drop | `lib/pinchflat.ex` | `-` | — | empty docs module |
| drop | `lib/pinchflat/boot/post_job_startup_tasks.ex` | `-` | — | empty GenServer |
| drop | `lib/pinchflat/fast_indexing/youtube_behaviour.ex` | `-` | — | becomes an interface declared in youtube_api.go |
| drop | `lib/pinchflat/http/http_behaviour.ex` | `-` | — | becomes an interface declared in http_client.go |
| drop | `lib/pinchflat/lifecycle/notifications/apprise_command_runner.ex` | `-` | — | behaviour -> interface in command_runner.go |
| drop | `lib/pinchflat/lifecycle/user_scripts/user_script_command_runner.ex` | `-` | — | behaviour -> interface in command_runner.go |
| drop | `lib/pinchflat/mailer.ex` | `-` | — | Swoosh mailer is never called |
| drop | `lib/pinchflat/prom_ex.ex` | `-` | — | replaced by client_golang in W4 (metric names change) |
| drop | `lib/pinchflat/release.ex` | `-` | — | replaced by the embedded migrator (W0) |
| drop | `lib/pinchflat/yt_dlp/yt_dlp_command_runner.ex` | `-` | — | behaviour -> interface in command_runner.go |
| drop | `lib/pinchflat_web.ex` | `-` | — | Phoenix `use` macros |
| drop | `lib/pinchflat_web/controllers/error_json.ex` | `-` | `test/pinchflat_web/controllers/error_json_test.exs` | Phoenix JSON error view; net/http default |
| drop | `lib/pinchflat_web/controllers/sources/source_html/index_table_live.ex` | `-` | `test/pinchflat_web/controllers/sources/index_table_live_test.exs` | dead code: PinchflatWeb.Sources.IndexTableLive is never referenced (the live one is SourceLive.IndexTableLive) |
| drop | `lib/pinchflat_web/gettext.ex` | `-` | — | 3 English strings; inline them |
| drop | `lib/pinchflat_web/telemetry.ex` | `-` | — | Phoenix/VM telemetry; replaced by W4 metrics |

Counts: W0: 1, W1: 29, W2: 17, W3: 19, W4: 69, drop: 15
