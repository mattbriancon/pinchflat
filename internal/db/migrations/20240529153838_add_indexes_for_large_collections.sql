CREATE INDEX "media_items_pending_and_downloaded_index" ON "media_items" ("source_id", "media_filepath", "uploaded_at", "prevent_download", "livestream", "short_form_content", "title");
-- +statement
