ALTER TABLE "media_items" ADD COLUMN "upload_date_index" INTEGER DEFAULT 0 NOT NULL;
-- +statement
CREATE INDEX "media_items_upload_date_index" ON "media_items" ("upload_date");
-- +statement
