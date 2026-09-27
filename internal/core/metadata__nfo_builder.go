package core

import (
	"context"
	"fmt"
	"time"
)

// NfoBuilderBuildAndStoreForMediaItem/2
func NfoBuilderBuildAndStoreForMediaItem(nfoFilepath string, metadata map[string]any) (string, error) {
	nfo := buildForMediaItem(nfoFilepath, metadata)

	ctx := context.Background()
	if err := FilesystemUtilsWriteP(ctx, nfoFilepath, nfo, []string{}); err != nil {
		return "", err
	}

	return nfoFilepath, nil
}

// NfoBuilderBuildAndStoreForSource/2
func NfoBuilderBuildAndStoreForSource(filepath string, metadata map[string]any) (string, error) {
	nfo := buildForSource(metadata)

	ctx := context.Background()
	if err := FilesystemUtilsWriteP(ctx, filepath, nfo, []string{}); err != nil {
		return "", err
	}

	return filepath, nil
}

func buildForMediaItem(nfoFilepath string, metadata map[string]any) string {
	var uploadDate time.Time
	var season, episode string

	uploadDateStr, ok := metadata["upload_date"].(string)
	if ok {
		uploadDate, _ = MetadataFileHelpersParseUploadDate(uploadDateStr)
	}

	// Determine season and episode number
	seasonStr, episodeStr, err := MetadataFileHelpersSeasonAndEpisodeFromMediaFilepath(nfoFilepath)
	if err == nil {
		season = seasonStr
		episode = episodeStr
	} else {
		// Fallback to upload date
		season = fmt.Sprintf("%d", uploadDate.Year())
		episode = uploadDate.Format("0102")
	}

	title := getStringFromMetadata(metadata, "title")
	uploader := getStringFromMetadata(metadata, "uploader")
	id := getStringFromMetadata(metadata, "id")
	description := getStringFromMetadata(metadata, "description")
	aired := uploadDate.Format("2006-01-02")

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes" ?>
<episodedetails>
  <title>%s</title>
  <showtitle>%s</showtitle>
  <uniqueid type="youtube" default="true">%s</uniqueid>
  <plot>%s</plot>
  <aired>%s</aired>
  <season>%s</season>
  <episode>%s</episode>
  <genre>YouTube</genre>
</episodedetails>
`, XmlUtilsSafe(title), XmlUtilsSafe(uploader), XmlUtilsSafe(id), XmlUtilsSafe(description), XmlUtilsSafe(aired), XmlUtilsSafe(season), XmlUtilsSafe(episode))
}

func buildForSource(metadata map[string]any) string {
	title := getStringFromMetadata(metadata, "title")
	description := getStringFromMetadata(metadata, "description")
	id := getStringFromMetadata(metadata, "id")

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" standalone="yes" ?>
<tvshow>
  <title>%s</title>
  <plot>%s</plot>
  <uniqueid type="youtube" default="true">%s</uniqueid>
  <genre>YouTube</genre>
</tvshow>
`, XmlUtilsSafe(title), XmlUtilsSafe(description), XmlUtilsSafe(id))
}

func getStringFromMetadata(metadata map[string]any, key string) string {
	if val, ok := metadata[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
