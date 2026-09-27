ALTER TABLE "media_profiles" ADD COLUMN "sponsorblock_behaviour" TEXT DEFAULT 'disabled';
-- +statement
ALTER TABLE "media_profiles" ADD COLUMN "sponsorblock_categories" TEXT DEFAULT ('[]');
-- +statement
