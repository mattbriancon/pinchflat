ALTER TABLE "sources" ADD COLUMN "cookie_behaviour" TEXT DEFAULT 'disabled' NOT NULL;
-- +statement
UPDATE sources SET cookie_behaviour = 'all_operations' WHERE use_cookies = TRUE;
-- +statement
ALTER TABLE "sources" DROP COLUMN "use_cookies";
-- +statement
