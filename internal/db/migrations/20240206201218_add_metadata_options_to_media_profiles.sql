ALTER TABLE "media_profiles" ADD COLUMN "download_metadata" INTEGER DEFAULT true NOT NULL;
-- +statement
ALTER TABLE "media_profiles" ADD COLUMN "embed_metadata" INTEGER DEFAULT true NOT NULL;
-- +statement
ALTER TABLE "media_items" ADD COLUMN "metadata_filepath" TEXT;
-- +statement
