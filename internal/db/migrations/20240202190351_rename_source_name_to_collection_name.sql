ALTER TABLE "sources" RENAME COLUMN "name" TO "collection_name";
-- +statement
ALTER TABLE "sources" ADD COLUMN "friendly_name" TEXT;
-- +statement
