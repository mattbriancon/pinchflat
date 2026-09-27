ALTER TABLE "media_items" ADD COLUMN "culled_at" TEXT;
-- +statement
ALTER TABLE "media_items" ADD COLUMN "prevent_culling" INTEGER DEFAULT false;
-- +statement
