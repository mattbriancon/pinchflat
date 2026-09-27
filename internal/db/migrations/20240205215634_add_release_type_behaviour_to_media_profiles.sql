ALTER TABLE "media_profiles" ADD COLUMN "shorts_behaviour" TEXT DEFAULT 'include' NOT NULL;
-- +statement
ALTER TABLE "media_profiles" ADD COLUMN "livestream_behaviour" TEXT DEFAULT 'include' NOT NULL;
-- +statement
