CREATE TABLE "sources" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL, "collection_id" TEXT NOT NULL, "collection_type" TEXT NOT NULL, "original_url" TEXT NOT NULL, "media_profile_id" INTEGER NOT NULL CONSTRAINT "sources_media_profile_id_fkey" REFERENCES "media_profiles"("id") ON DELETE RESTRICT, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL);
-- +statement
CREATE INDEX "sources_media_profile_id_index" ON "sources" ("media_profile_id");
-- +statement
CREATE UNIQUE INDEX "sources_collection_id_media_profile_id_index" ON "sources" ("collection_id", "media_profile_id");
-- +statement
