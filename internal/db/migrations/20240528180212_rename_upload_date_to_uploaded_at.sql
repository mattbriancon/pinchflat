ALTER TABLE "media_items" RENAME COLUMN "upload_date" TO "uploaded_at";
-- +statement
UPDATE media_items
  SET uploaded_at = uploaded_at || 'T00:00:00';
-- +statement
