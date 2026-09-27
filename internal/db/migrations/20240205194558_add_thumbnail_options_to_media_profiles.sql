ALTER TABLE "media_profiles" ADD COLUMN "download_thumbnail" INTEGER DEFAULT true NOT NULL;
-- +statement
ALTER TABLE "media_profiles" ADD COLUMN "embed_thumbnail" INTEGER DEFAULT true NOT NULL;
-- +statement
ALTER TABLE "media_items" ADD COLUMN "thumbnail_filepath" TEXT;
-- +statement
