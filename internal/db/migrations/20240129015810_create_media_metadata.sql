CREATE TABLE "media_metadata" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "client_response" JSON NOT NULL, "media_item_id" INTEGER NOT NULL CONSTRAINT "media_metadata_media_item_id_fkey" REFERENCES "media_items"("id") ON DELETE CASCADE, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL);
-- +statement
CREATE UNIQUE INDEX "media_metadata_media_item_id_index" ON "media_metadata" ("media_item_id");
-- +statement
