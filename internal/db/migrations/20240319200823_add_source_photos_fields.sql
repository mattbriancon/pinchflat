ALTER TABLE "sources" ADD COLUMN "fanart_filepath" TEXT;
-- +statement
ALTER TABLE "sources" ADD COLUMN "poster_filepath" TEXT;
-- +statement
ALTER TABLE "sources" ADD COLUMN "banner_filepath" TEXT;
-- +statement
ALTER TABLE "media_profiles" ADD COLUMN "download_source_images" INTEGER DEFAULT false NOT NULL;
-- +statement
