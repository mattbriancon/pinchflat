package coretest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/db/dbtest"
)

// Now returns the current UTC time.
func Now() time.Time {
	return time.Now().UTC()
}

// NowPlus adds an offset to the current UTC time.
// unit is "minute" or "minutes" or "day" or "days".
func NowPlus(offset int, unit string) time.Time {
	switch unit {
	case "minute", "minutes":
		return Now().Add(time.Duration(offset) * time.Minute)
	case "day", "days":
		return Now().Add(time.Duration(offset) * 24 * time.Hour)
	default:
		return Now()
	}
}

// NowMinus subtracts an offset from the current UTC time.
// unit is "minute" or "minutes" or "day" or "days".
func NowMinus(offset int, unit string) time.Time {
	return NowPlus(-offset, unit)
}

// RenderMetadata reads a JSON metadata file from test/support/files directory.
func RenderMetadata(metadataName string) (string, error) {
	jsonFilepath := filepath.Join(
		dbtest.RepoRoot(),
		"test",
		"support",
		"files",
		metadataName+".json",
	)

	content, err := os.ReadFile(jsonFilepath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// RenderParsedMetadata reads and decodes a JSON metadata file.
func RenderParsedMetadata(metadataName string) (map[string]interface{}, error) {
	content, err := RenderMetadata(metadataName)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal([]byte(content), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// CreatePlatformDirectories creates all the media, metadata, extras and tmpfile directories.
func CreatePlatformDirectories(a *core.App) error {
	for _, dir := range []string{
		a.Config.MediaDirectory,
		a.Config.MetadataDirectory,
		a.Config.ExtrasDirectory,
		a.Config.TmpfileDirectory,
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}
