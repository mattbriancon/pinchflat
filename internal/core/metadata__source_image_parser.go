package core

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

// toFloat coerces a numeric or numeric-string value to a float64.
func toFloat(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return 0, false
		}
		rv = rv.Elem()
	}
	switch {
	case rv.CanInt():
		return float64(rv.Int()), true
	case rv.CanUint():
		return float64(rv.Uint()), true
	case rv.CanFloat():
		return rv.Float(), true
	case rv.Kind() == reflect.String:
		f, err := strconv.ParseFloat(strings.TrimSpace(rv.String()), 64)
		return f, err == nil
	}
	return 0, false
}

// SourceImageParserStoreSourceImages/2
func SourceImageParserStoreSourceImages(baseDirectory string, sourceMetadata map[string]any) (map[string]string, error) {
	thumbnails := getThumbnailsFromMetadata(sourceMetadata)

	// Filter images with filepath
	var filteredImages []map[string]any
	for _, img := range thumbnails {
		if imgMap, ok := img.(map[string]any); ok {
			if filepath, ok := imgMap["filepath"].(string); ok && filepath != "" {
				filteredImages = append(filteredImages, imgMap)
			}
		}
	}

	// Select useful images
	labelledImages := selectUsefulImages(filteredImages, sourceMetadata)

	// Move images and build result map
	result := make(map[string]string)

	for key, attrs := range labelledImages {
		finalPathmap := moveImage(key, attrs, baseDirectory)
		for k, v := range finalPathmap {
			result[k] = v
		}
	}

	return result, nil
}

type imageAttrs struct {
	attributeName   string
	finalFilename   string
	currentFilepath string
}

func getThumbnailsFromMetadata(sourceMetadata map[string]any) []any {
	if thumbnails, ok := sourceMetadata["thumbnails"].([]any); ok {
		return thumbnails
	}
	return []any{}
}

func selectUsefulImages(images []map[string]any, sourceMetadata map[string]any) map[string]imageAttrs {
	labelledImages := make(map[string]imageAttrs)

	// Process each image for avatar and banner
	for _, imageMap := range images {
		if id, ok := imageMap["id"].(string); ok {
			if id == "avatar_uncropped" {
				if filepath, ok := imageMap["filepath"].(string); ok {
					labelledImages["poster"] = imageAttrs{
						attributeName:   "poster_filepath",
						finalFilename:   "poster",
						currentFilepath: filepath,
					}
				}
			} else if id == "banner_uncropped" {
				if filepath, ok := imageMap["filepath"].(string); ok {
					labelledImages["fanart"] = imageAttrs{
						attributeName:   "fanart_filepath",
						finalFilename:   "fanart",
						currentFilepath: filepath,
					}
				}
			}
		}
	}

	// Add fallback poster if needed
	labelledImages = addFallbackPoster(labelledImages, sourceMetadata)

	// Add best banner
	bestBanner := determineBestBanner(images)
	if bestBanner != "" {
		labelledImages["banner"] = imageAttrs{
			attributeName:   "banner_filepath",
			finalFilename:   "banner",
			currentFilepath: bestBanner,
		}
	}

	return labelledImages
}

func addFallbackPoster(images map[string]imageAttrs, sourceMetadata map[string]any) map[string]imageAttrs {
	// If poster is already set, return as-is
	if _, exists := images["poster"]; exists {
		return images
	}

	// Try to get poster from entries
	if entries, ok := sourceMetadata["entries"].([]any); ok && len(entries) > 0 {
		if firstEntry, ok := entries[0].(map[string]any); ok {
			if thumbnail := getPosterFromEntry(firstEntry); thumbnail != "" {
				images["poster"] = imageAttrs{
					attributeName:   "poster_filepath",
					finalFilename:   "poster",
					currentFilepath: thumbnail,
				}
			}
		}
	}

	return images
}

func getPosterFromEntry(entry map[string]any) string {
	if thumbnails, ok := entry["thumbnails"].([]any); ok {
		// Reverse order and find first with filepath
		for i := len(thumbnails) - 1; i >= 0; i-- {
			if thumb, ok := thumbnails[i].(map[string]any); ok {
				if filepath, ok := thumb["filepath"].(string); ok && filepath != "" {
					return filepath
				}
			}
		}
	}
	return ""
}

func determineBestBanner(images []map[string]any) string {
	var candidates []map[string]any

	// Filter images with width and height (numbers come back as json.Number,
	// not float64, when decoded via store.DecodeJSON).
	for _, img := range images {
		if width, okW := img["width"]; okW {
			if height, okH := img["height"]; okH {
				if _, ok := toFloat(width); ok {
					if _, ok := toFloat(height); ok {
						candidates = append(candidates, img)
					}
				}
			}
		}
	}

	// Find best candidate with aspect ratio > 3
	var bestCandidate map[string]any
	maxWidth := 0.0

	for _, candidate := range candidates {
		width, _ := toFloat(candidate["width"])
		height, _ := toFloat(candidate["height"])

		if height > 0 && width/height > 3 && width > maxWidth {
			maxWidth = width
			bestCandidate = candidate
		}
	}

	if bestCandidate != nil {
		if filepath, ok := bestCandidate["filepath"].(string); ok {
			return filepath
		}
	}

	return ""
}

func moveImage(key string, attrs imageAttrs, baseDirectory string) map[string]string {
	extension := filepath.Ext(attrs.currentFilepath)
	finalFilepath := filepath.Join(baseDirectory, fmt.Sprintf("%s%s", attrs.finalFilename, extension))

	if err := fsutil.CopyFile(attrs.currentFilepath, finalFilepath); err != nil {
		return map[string]string{}
	}

	return map[string]string{attrs.attributeName: finalFilepath}
}
