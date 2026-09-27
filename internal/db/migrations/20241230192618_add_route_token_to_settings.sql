ALTER TABLE "settings" ADD COLUMN "route_token" TEXT DEFAULT 'tmp-token' NOT NULL;
-- +statement
UPDATE settings SET route_token = gen_random_uuid();
-- +statement
