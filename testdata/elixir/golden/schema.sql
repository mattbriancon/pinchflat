CREATE TABLE sqlean_define(name text primary key, type text, body text);
CREATE TABLE IF NOT EXISTS "schema_migrations" ("version" INTEGER PRIMARY KEY, "inserted_at" TEXT);
CREATE TABLE IF NOT EXISTS "media_profiles" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL, "output_path_template" TEXT NOT NULL, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL, "download_subs" INTEGER DEFAULT true NOT NULL, "download_auto_subs" INTEGER DEFAULT true NOT NULL, "embed_subs" INTEGER DEFAULT true NOT NULL, "sub_langs" TEXT DEFAULT 'en' NOT NULL, "download_thumbnail" INTEGER DEFAULT true NOT NULL, "embed_thumbnail" INTEGER DEFAULT true NOT NULL, "shorts_behaviour" TEXT DEFAULT 'include' NOT NULL, "livestream_behaviour" TEXT DEFAULT 'include' NOT NULL, "download_metadata" INTEGER DEFAULT true NOT NULL, "embed_metadata" INTEGER DEFAULT true NOT NULL, "preferred_resolution" TEXT DEFAULT '1080p' NOT NULL, "download_nfo" INTEGER DEFAULT false NOT NULL, "download_source_images" INTEGER DEFAULT false NOT NULL, "sponsorblock_behaviour" TEXT DEFAULT 'disabled', "sponsorblock_categories" TEXT DEFAULT ('[]'), "redownload_delay_days" INTEGER, "marked_for_deletion_at" TEXT, "media_container" TEXT, "audio_track" TEXT);
CREATE TABLE sqlite_sequence(name,seq);
CREATE TABLE IF NOT EXISTS "sources" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "collection_name" TEXT NOT NULL, "collection_id" TEXT NOT NULL, "collection_type" TEXT NOT NULL, "original_url" TEXT NOT NULL, "media_profile_id" INTEGER NOT NULL CONSTRAINT "sources_media_profile_id_fkey" REFERENCES "media_profiles"("id") ON DELETE RESTRICT, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL, "index_frequency_minutes" INTEGER DEFAULT 1440 NOT NULL, "download_media" INTEGER DEFAULT true NOT NULL, "last_indexed_at" TEXT, "custom_name" TEXT NOT NULL, "fast_index" INTEGER DEFAULT false NOT NULL, "download_cutoff_date" TEXT, "nfo_filepath" TEXT, "series_directory" TEXT, "fanart_filepath" TEXT, "poster_filepath" TEXT, "banner_filepath" TEXT, "title_filter_regex" TEXT, "uuid" TEXT, "description" TEXT, "retention_period_days" INTEGER, "output_path_template_override" TEXT, "marked_for_deletion_at" TEXT, "min_duration_seconds" INTEGER, "max_duration_seconds" INTEGER, "enabled" INTEGER DEFAULT true NOT NULL, "cookie_behaviour" TEXT DEFAULT 'disabled' NOT NULL);
CREATE TABLE IF NOT EXISTS "media_items" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "media_id" TEXT NOT NULL, "title" TEXT, "media_filepath" TEXT, "source_id" INTEGER NOT NULL CONSTRAINT "media_items_source_id_fkey" REFERENCES "sources"("id") ON DELETE RESTRICT, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL, "subtitle_filepaths" TEXT DEFAULT ('[]'), "thumbnail_filepath" TEXT, "metadata_filepath" TEXT, "livestream" INTEGER DEFAULT false NOT NULL, "original_url" TEXT NOT NULL, "media_downloaded_at" TEXT, "details_updated_at" TEXT, "description" TEXT, "media_size_bytes" INTEGER, "short_form_content" INTEGER DEFAULT false NOT NULL, "uploaded_at" TEXT DEFAULT '1970-01-01' NOT NULL, "nfo_filepath" TEXT, "uuid" TEXT, "duration_seconds" INTEGER, "prevent_download" INTEGER DEFAULT false NOT NULL, "culled_at" TEXT, "prevent_culling" INTEGER DEFAULT false, "media_redownloaded_at" TEXT, "upload_date_index" INTEGER DEFAULT 0 NOT NULL, "playlist_index" INTEGER DEFAULT 0 NOT NULL, "predicted_media_filepath" TEXT, "last_error" TEXT);
CREATE TABLE IF NOT EXISTS "oban_jobs" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "state" TEXT DEFAULT 'available' NOT NULL, "queue" TEXT DEFAULT 'default' NOT NULL, "worker" TEXT NOT NULL, "args" JSON DEFAULT ('{}') NOT NULL, "meta" JSON DEFAULT ('{}') NOT NULL, "tags" JSON DEFAULT ('[]') NOT NULL, "errors" JSON DEFAULT ('[]') NOT NULL, "attempt" INTEGER DEFAULT 0 NOT NULL, "max_attempts" INTEGER DEFAULT 20 NOT NULL, "priority" INTEGER DEFAULT 0 NOT NULL, "inserted_at" TEXT DEFAULT CURRENT_TIMESTAMP NOT NULL, "scheduled_at" TEXT DEFAULT CURRENT_TIMESTAMP NOT NULL, "attempted_at" TEXT, "attempted_by" JSON DEFAULT ('[]') NOT NULL, "cancelled_at" TEXT, "completed_at" TEXT, "discarded_at" TEXT);
CREATE TABLE IF NOT EXISTS "tasks" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "job_id" INTEGER NOT NULL CONSTRAINT "tasks_job_id_fkey" REFERENCES "oban_jobs"("id") ON DELETE CASCADE, "source_id" INTEGER NULL CONSTRAINT "tasks_source_id_fkey" REFERENCES "sources"("id") ON DELETE RESTRICT, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL, "media_item_id" INTEGER NULL CONSTRAINT "tasks_media_item_id_fkey" REFERENCES "media_items"("id") ON DELETE RESTRICT);
CREATE TABLE IF NOT EXISTS "media_metadata" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "media_item_id" INTEGER NOT NULL CONSTRAINT "media_metadata_media_item_id_fkey" REFERENCES "media_items"("id") ON DELETE CASCADE, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL, "metadata_filepath" TEXT NOT NULL, "thumbnail_filepath" TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS "settings_backup" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL, "value" TEXT NOT NULL, "datatype" TEXT NOT NULL, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL);
CREATE TABLE IF NOT EXISTS "source_metadata" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "metadata_filepath" TEXT NOT NULL, "source_id" INTEGER NOT NULL CONSTRAINT "source_metadata_source_id_fkey" REFERENCES "sources"("id") ON DELETE CASCADE, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL, "fanart_filepath" TEXT, "poster_filepath" TEXT, "banner_filepath" TEXT);
CREATE TABLE IF NOT EXISTS "settings" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "onboarding" INTEGER DEFAULT true NOT NULL, "pro_enabled" INTEGER DEFAULT false NOT NULL, "yt_dlp_version" TEXT, "apprise_server" TEXT, "apprise_version" TEXT, "video_codec_preference" TEXT DEFAULT 'avc' NOT NULL, "audio_codec_preference" TEXT DEFAULT 'm4a' NOT NULL, "youtube_api_key" TEXT, "route_token" TEXT DEFAULT 'tmp-token' NOT NULL, "extractor_sleep_interval_seconds" NUMBER DEFAULT 0 NOT NULL, "download_throughput_limit" TEXT, "restrict_filenames" INTEGER DEFAULT false);
CREATE TABLE IF NOT EXISTS 'media_items_search_index_data'(id INTEGER PRIMARY KEY, block BLOB);
CREATE TABLE IF NOT EXISTS 'media_items_search_index_idx'(segid, term, pgno, PRIMARY KEY(segid, term)) WITHOUT ROWID;
CREATE TABLE IF NOT EXISTS 'media_items_search_index_content'(id INTEGER PRIMARY KEY, c0, c1);
CREATE TABLE IF NOT EXISTS 'media_items_search_index_docsize'(id INTEGER PRIMARY KEY, sz BLOB);
CREATE TABLE IF NOT EXISTS 'media_items_search_index_config'(k PRIMARY KEY, v) WITHOUT ROWID;
CREATE UNIQUE INDEX "media_profiles_name_index" ON "media_profiles" ("name");
CREATE INDEX "sources_media_profile_id_index" ON "sources" ("media_profile_id");
CREATE INDEX "media_items_source_id_index" ON "media_items" ("source_id");
CREATE UNIQUE INDEX "media_items_media_id_source_id_index" ON "media_items" ("media_id", "source_id");
CREATE INDEX "oban_jobs_state_queue_priority_scheduled_at_id_index" ON "oban_jobs" ("state", "queue", "priority", "scheduled_at", "id");
CREATE INDEX "tasks_job_id_index" ON "tasks" ("job_id");
CREATE INDEX "tasks_source_id_index" ON "tasks" ("source_id");
CREATE UNIQUE INDEX "media_metadata_media_item_id_index" ON "media_metadata" ("media_item_id");
CREATE INDEX "tasks_media_item_id_index" ON "tasks" ("media_item_id");
CREATE UNIQUE INDEX "settings_name_index" ON "settings_backup" ("name");
CREATE UNIQUE INDEX "source_metadata_source_id_index" ON "source_metadata" ("source_id");
CREATE UNIQUE INDEX "sources_uuid_index" ON "sources" ("uuid");
CREATE UNIQUE INDEX "media_items_uuid_index" ON "media_items" ("uuid");
CREATE UNIQUE INDEX sources_collection_id_media_profile_id_title_filter_regex_index ON sources (
    collection_id,
    media_profile_id,
    IFNULL(title_filter_regex, '')
  );
CREATE INDEX "media_items_media_downloaded_at_index" ON "media_items" ("media_downloaded_at");
CREATE INDEX "media_items_media_redownloaded_at_index" ON "media_items" ("media_redownloaded_at");
CREATE INDEX "media_items_uploaded_at_index" ON "media_items" ("uploaded_at");
CREATE INDEX "media_items_pending_and_downloaded_index" ON "media_items" ("source_id", "media_filepath", "uploaded_at", "prevent_download", "livestream", "short_form_content", "title");
CREATE VIRTUAL TABLE media_items_search_index USING fts5(
    title,
    description,
    tokenize=trigram
  )
/* media_items_search_index(title,description) */;
CREATE TRIGGER media_items_search_index_insert AFTER INSERT ON media_items BEGIN
    INSERT INTO media_items_search_index(
      rowid,
      title,
      description
    )
    VALUES(
      new.id,
      new.title,
      new.description
    );
  END;
CREATE TRIGGER media_items_search_index_update AFTER UPDATE ON media_items BEGIN
    UPDATE media_items_search_index SET
      title = new.title,
      description = new.description
    WHERE
      rowid = old.id;
  END;
CREATE TRIGGER media_items_search_index_delete AFTER DELETE ON media_items BEGIN
    DELETE FROM media_items_search_index WHERE rowid = old.id;
  END;
