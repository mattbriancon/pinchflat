package coretest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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

// RenderMetadata reads a JSON metadata file from testdata/support/files.
func RenderMetadata(metadataName string) (string, error) {
	jsonFilepath := filepath.Join(
		dbtest.RepoRoot(),
		"testdata",
		"support",
		"files",
		metadataName+".json",
	)

	content, err := os.ReadFile(jsonFilepath)
	if err != nil {
		return "", err
	}

	// The fixtures were recorded in the Elixir test container, so their
	// filepaths start with /app/tmp. Point them at a writable directory:
	// CI runs as a normal user and can't create /app.
	return strings.ReplaceAll(string(content), "/app/tmp/", fixtureTmpDir()+"/"), nil
}

var (
	fixtureTmpOnce sync.Once
	fixtureTmp     string
)

// fixtureTmpDir is a per-process directory standing in for the fixtures'
// /app/tmp.
func fixtureTmpDir() string {
	fixtureTmpOnce.Do(func() {
		dir, err := os.MkdirTemp("", "pinchflat-fixtures-")
		if err != nil {
			panic(err)
		}
		fixtureTmp = dir
	})
	return fixtureTmp
}

// RenderMetadataWithFixedPaths reads a JSON metadata file and fixes hardcoded paths to work in the test environment.
func RenderMetadataWithFixedPaths(metadataName string) (string, error) {
	jsonFilepath := filepath.Join(
		dbtest.RepoRoot(),
		"testdata",
		"support",
		"files",
		metadataName+".json",
	)

	content, err := os.ReadFile(jsonFilepath)
	if err != nil {
		return "", err
	}

	// Parse the JSON and fix the thumbnail filepaths
	var data map[string]any
	if err := json.Unmarshal(content, &data); err != nil {
		return "", err
	}

	// Fix thumbnail paths
	if thumbnails, ok := data["thumbnails"].([]any); ok {
		testPhotosDir := filepath.Join(dbtest.RepoRoot(), "testdata", "support", "files", "channel_photos")
		for _, thumb := range thumbnails {
			if thumbMap, ok := thumb.(map[string]any); ok {
				if fp, ok := thumbMap["filepath"].(string); ok {
					// Extract just the filename and replace with test path
					filename := filepath.Base(fp)
					thumbMap["filepath"] = filepath.Join(testPhotosDir, filename)
				}
			}
		}
	}

	// Encode back to JSON string
	result, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	return string(result), nil
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
