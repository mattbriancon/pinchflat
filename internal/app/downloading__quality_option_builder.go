package app

import (
	"context"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/store"
)

// QualityOptionBuilder builds quality-related options for yt-dlp.

// QualityOptionBuilderBuild/1
func (a *App) QualityOptionBuilderBuild(ctx context.Context, mediaProfile *store.MediaProfile) store.KW {
	if mediaProfile.PreferredResolution == store.MediaProfilePreferredResolutionAudio {
		return qualityOptionBuilderBuildAudio(ctx, a, mediaProfile)
	}
	return qualityOptionBuilderBuildVideo(ctx, a, mediaProfile)
}

func qualityOptionBuilderBuildAudio(ctx context.Context, a *App, mediaProfile *store.MediaProfile) store.KW {
	audioCodec, _ := a.GetSetting(ctx, "audio_codec_preference")
	audioCodecStr := audioCodec.(string)

	container := "best"
	if mediaProfile.MediaContainer != nil {
		container = *mediaProfile.MediaContainer
	}

	return store.KW{
		store.Flag("extract_audio"),
		store.Opt("format_sort", "+acodec:"+audioCodecStr),
		store.Opt("audio_format", container),
		store.Opt("format", qualityOptionBuilderBuildFormatString(mediaProfile)),
	}
}

func qualityOptionBuilderBuildVideo(ctx context.Context, a *App, mediaProfile *store.MediaProfile) store.KW {
	videoCodec, _ := a.GetSetting(ctx, "video_codec_preference")
	audioCodec, _ := a.GetSetting(ctx, "audio_codec_preference")
	videoCodecStr := videoCodec.(string)
	audioCodecStr := audioCodec.(string)

	// Extract resolution digits from the string (e.g., "480p" -> "480")
	resolutionStr := string(mediaProfile.PreferredResolution)
	resolutionDigits := strings.TrimSuffix(resolutionStr, "p")

	container := "mp4"
	if mediaProfile.MediaContainer != nil {
		container = *mediaProfile.MediaContainer
	}

	return store.KW{
		store.Opt("remux_video", container),
		store.Opt("format_sort", "res:"+resolutionDigits+",+codec:"+videoCodecStr+":"+audioCodecStr),
		store.Opt("format", qualityOptionBuilderBuildFormatString(mediaProfile)),
	}
}

func qualityOptionBuilderBuildFormatString(mediaProfile *store.MediaProfile) string {
	if mediaProfile.PreferredResolution == store.MediaProfilePreferredResolutionAudio {
		if mediaProfile.AudioTrack != nil {
			return "bestaudio[" + qualityOptionBuilderBuildFormatModifier(*mediaProfile.AudioTrack) + "]/bestaudio/best"
		}
		return "bestaudio/best"
	}

	if mediaProfile.AudioTrack != nil {
		return "bestvideo+bestaudio[" + qualityOptionBuilderBuildFormatModifier(*mediaProfile.AudioTrack) + "]/bestvideo*+bestaudio/best"
	}
	return "bestvideo*+bestaudio/best"
}

func qualityOptionBuilderBuildFormatModifier(audioTrack string) string {
	if audioTrack == "original" {
		return "format_note*=original"
	}
	if audioTrack == "default" {
		return "format_note*='(default)'"
	}
	// Language code case
	return "language^=" + audioTrack
}
