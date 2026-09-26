ALTER TABLE "sources" DROP COLUMN "friendly_name";
-- +statement
ALTER TABLE "sources" ADD COLUMN "friendly_name" TEXT NOT NULL;
-- +statement
