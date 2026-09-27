ALTER TABLE "media_metadata" ADD COLUMN "metadata_filepath" TEXT NOT NULL;
-- +statement
ALTER TABLE "media_metadata" ADD COLUMN "thumbnail_filepath" TEXT NOT NULL;
-- +statement
ALTER TABLE "media_metadata" DROP COLUMN "client_response";
-- +statement
