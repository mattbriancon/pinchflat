ALTER TABLE "settings" DROP COLUMN "video_codec_preference";
-- +statement
ALTER TABLE "settings" DROP COLUMN "audio_codec_preference";
-- +statement
ALTER TABLE "settings" ADD COLUMN "video_codec_preference" TEXT DEFAULT 'avc' NOT NULL;
-- +statement
ALTER TABLE "settings" ADD COLUMN "audio_codec_preference" TEXT DEFAULT 'm4a' NOT NULL;
-- +statement
