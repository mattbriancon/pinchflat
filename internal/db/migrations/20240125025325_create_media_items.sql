CREATE TABLE "media_items" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "media_id" TEXT NOT NULL, "title" TEXT, "video_filepath" TEXT, "source_id" INTEGER NOT NULL CONSTRAINT "media_items_source_id_fkey" REFERENCES "sources"("id") ON DELETE RESTRICT, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL);
-- +statement
CREATE INDEX "media_items_source_id_index" ON "media_items" ("source_id");
-- +statement
CREATE UNIQUE INDEX "media_items_media_id_source_id_index" ON "media_items" ("media_id", "source_id");
-- +statement
