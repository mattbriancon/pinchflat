CREATE TABLE "settings" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "name" TEXT NOT NULL, "value" TEXT NOT NULL, "datatype" TEXT NOT NULL, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL);
-- +statement
CREATE UNIQUE INDEX "settings_name_index" ON "settings" ("name");
-- +statement
