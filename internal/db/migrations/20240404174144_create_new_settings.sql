CREATE TABLE "settings" ("id" INTEGER PRIMARY KEY AUTOINCREMENT, "onboarding" INTEGER DEFAULT true NOT NULL, "pro_enabled" INTEGER DEFAULT false NOT NULL, "yt_dlp_version" TEXT);
-- +statement
INSERT INTO settings (onboarding, pro_enabled, yt_dlp_version) VALUES (true, false, NULL);
-- +statement
UPDATE settings
  SET onboarding = COALESCE((SELECT value = 'true' FROM settings_backup WHERE name = 'onboarding'), true);
-- +statement
UPDATE settings
  SET pro_enabled = COALESCE((SELECT value = 'true' FROM settings_backup WHERE name = 'pro_enabled'), false);
-- +statement
