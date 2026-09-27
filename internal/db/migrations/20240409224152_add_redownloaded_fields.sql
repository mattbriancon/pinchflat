ALTER TABLE "media_profiles" ADD COLUMN "redownload_delay_days" INTEGER;
-- +statement
ALTER TABLE "media_items" ADD COLUMN "media_redownloaded_at" TEXT;
-- +statement
