CREATE TABLE "source_metadata" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "metadata_filepath" TEXT NOT NULL, "source_id" INTEGER NOT NULL CONSTRAINT "source_metadata_source_id_fkey" REFERENCES "sources"("id") ON DELETE CASCADE, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL);
-- +statement
CREATE UNIQUE INDEX "source_metadata_source_id_index" ON "source_metadata" ("source_id");
-- +statement
