ALTER TABLE "media_items" ADD COLUMN "livestream" INTEGER DEFAULT false NOT NULL;
-- +statement
ALTER TABLE "media_items" ADD COLUMN "original_url" TEXT NOT NULL;
-- +statement
