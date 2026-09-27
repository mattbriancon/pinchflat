UPDATE sources SET uuid = gen_random_uuid() WHERE uuid IS NULL;
-- +statement
UPDATE media_items SET uuid = gen_random_uuid() WHERE uuid IS NULL;
-- +statement
