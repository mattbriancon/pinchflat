ALTER TABLE "media_items" ADD COLUMN "media_downloaded_at" TEXT;
-- +statement
ALTER TABLE "media_items" ADD COLUMN "details_updated_at" TEXT;
-- +statement
ALTER TABLE "sources" ADD COLUMN "last_indexed_at" TEXT;
-- +statement
