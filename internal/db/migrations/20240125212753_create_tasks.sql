CREATE TABLE "tasks" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "job_id" INTEGER NOT NULL CONSTRAINT "tasks_job_id_fkey" REFERENCES "oban_jobs"("id") ON DELETE CASCADE, "source_id" INTEGER NULL CONSTRAINT "tasks_source_id_fkey" REFERENCES "sources"("id") ON DELETE RESTRICT, "inserted_at" TEXT NOT NULL, "updated_at" TEXT NOT NULL);
-- +statement
CREATE INDEX "tasks_job_id_index" ON "tasks" ("job_id");
-- +statement
CREATE INDEX "tasks_source_id_index" ON "tasks" ("source_id");
-- +statement
