DROP INDEX "media_items_upload_date_index";
-- +statement
CREATE INDEX "media_items_uploaded_at_index" ON "media_items" ("uploaded_at");
-- +statement
