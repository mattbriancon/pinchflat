ALTER TABLE "settings" ADD COLUMN "video_codec_preference" TEXT DEFAULT ('[]');
-- +statement
ALTER TABLE "settings" ADD COLUMN "audio_codec_preference" TEXT DEFAULT ('[]');
-- +statement
