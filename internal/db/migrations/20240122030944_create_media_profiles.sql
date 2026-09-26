CREATE TABLE "media_profiles" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL, "output_path_template" TEXT NOT NULL, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL);
-- +statement
CREATE UNIQUE INDEX "media_profiles_name_index" ON "media_profiles" ("name");
-- +statement
