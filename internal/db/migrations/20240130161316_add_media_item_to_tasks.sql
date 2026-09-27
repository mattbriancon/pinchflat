ALTER TABLE "tasks" ADD COLUMN "media_item_id" INTEGER NULL CONSTRAINT "tasks_media_item_id_fkey" REFERENCES "media_items"("id") ON DELETE RESTRICT;
-- +statement
CREATE INDEX "tasks_media_item_id_index" ON "tasks" ("media_item_id");
-- +statement
