ALTER TABLE "media_profiles" ADD COLUMN "download_subs" INTEGER DEFAULT true NOT NULL;
-- +statement
ALTER TABLE "media_profiles" ADD COLUMN "download_auto_subs" INTEGER DEFAULT true NOT NULL;
-- +statement
ALTER TABLE "media_profiles" ADD COLUMN "embed_subs" INTEGER DEFAULT true NOT NULL;
-- +statement
ALTER TABLE "media_profiles" ADD COLUMN "sub_langs" TEXT DEFAULT 'en' NOT NULL;
-- +statement
