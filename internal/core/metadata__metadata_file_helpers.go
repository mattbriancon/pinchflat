package core

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/mattbriancon/pinchflat/internal/db"
	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

// MetadataFileHelpersMetadataDirectoryFor/1
func (a *App) MetadataFileHelpersMetadataDirectoryFor(ctx context.Context, databaseRecord any) (string, error) {
	// Get table name using TableName() method if available
	var tableName string
	if hasTableName, ok := databaseRecord.(interface{ TableName() string }); ok {
		tableName = hasTableName.TableName()
	} else {
		return "", fmt.Errorf("database record does not have TableName() method")
	}

	// Get ID using reflection
	recordValue := reflect.ValueOf(databaseRecord)
	// Dereference if pointer
	if recordValue.Kind() == reflect.Ptr {
		recordValue = recordValue.Elem()
	}

	idField := recordValue.FieldByName("ID")
	if !idField.IsValid() {
		return "", fmt.Errorf("database record does not have ID field")
	}

	id := idField.Int()

	return filepath.Join(
		a.Config.MetadataDirectory,
		tableName,
		fmt.Sprintf("%d", id),
	), nil
}

// MetadataFileHelpersCompressAndStoreMetadataFor/2
func (a *App) MetadataFileHelpersCompressAndStoreMetadataFor(ctx context.Context, databaseRecord any, metadataMap map[string]any) (string, error) {
	filepath, err := metadataFileHelpersGenerateFilepathFor(ctx, a, databaseRecord, "metadata.json.gz")
	if err != nil {
		return "", err
	}

	// Encode metadata to JSON
	jsonStr, err := db.EncodeJSON(metadataMap)
	if err != nil {
		return "", err
	}

	// Compress JSON
	var compressedBuf bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedBuf)
	if _, err := gzipWriter.Write([]byte(jsonStr)); err != nil {
		return "", err
	}
	if err := gzipWriter.Close(); err != nil {
		return "", err
	}

	// Write compressed file
	if err := fsutil.WriteFileAll(filepath, string(compressedBuf.Bytes())); err != nil {
		return "", err
	}

	return filepath, nil
}

// MetadataFileHelpersReadCompressedMetadata/1
func (a *App) MetadataFileHelpersReadCompressedMetadata(ctx context.Context, filepath string) (map[string]any, error) {
	// Read the file
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Decompress with gzip
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(gzipReader); err != nil {
		return nil, err
	}

	// Decode JSON
	var result map[string]any
	if err := DecodeJSON(buf.Bytes(), &result); err != nil {
		return nil, err
	}

	return result, nil
}

// MetadataFileHelpersDownloadAndStoreThumbnailFor/1
func (a *App) MetadataFileHelpersDownloadAndStoreThumbnailFor(ctx context.Context, mediaItemWithPreloads *MediaItem) (*string, error) {
	ytDlpFilepath, err := metadataFileHelpersGenerateFilepathFor(ctx, a, mediaItemWithPreloads, "thumbnail.%(ext)s")
	if err != nil {
		return nil, err
	}

	realFilepath, err := metadataFileHelpersGenerateFilepathFor(ctx, a, mediaItemWithPreloads, "thumbnail.jpg")
	if err != nil {
		return nil, err
	}

	commandOpts := KW{
		Opt("output", ytDlpFilepath),
	}

	addlOpts := KW{
		Opt("use_cookies", SourcesUseCookies(mediaItemWithPreloads.Source, "metadata")),
	}

	_, err = a.YtDlpMediaDownloadThumbnail(ctx, mediaItemWithPreloads.OriginalURL, commandOpts, addlOpts)
	if err != nil {
		return nil, nil
	}

	return &realFilepath, nil
}

// MetadataFileHelpersParseUploadDate/1
func MetadataFileHelpersParseUploadDate(uploadDate string) (time.Time, error) {
	if len(uploadDate) < 8 {
		return time.Time{}, fmt.Errorf("Invalid upload date: %s", uploadDate)
	}

	year := uploadDate[0:4]
	month := uploadDate[4:6]
	day := uploadDate[6:8]

	dt, err := time.Parse("2006-01-02T15:04:05Z", fmt.Sprintf("%s-%s-%sT00:00:00Z", year, month, day))
	if err != nil {
		return time.Time{}, fmt.Errorf("Invalid upload date: %s", uploadDate)
	}

	return dt, nil
}

// MetadataFileHelpersSeriesDirectoryFromMediaFilepath/1
func MetadataFileHelpersSeriesDirectoryFromMediaFilepath(mediaFilepath string) (string, error) {
	// Matches "s" or "season" (case-insensitive)
	// followed by an optional non-word character (. or _ or <space>, etc)
	// followed by at least one digit
	// followed immediately by the end of the string
	seasonRegex := regexp.MustCompile(`(?i)^s(eason)?(\W|_)?\d{1,}$`)

	// Split path into components
	parts := strings.Split(mediaFilepath, "/")

	var directoryAcc []string

	for _, part := range parts {
		if seasonRegex.MatchString(part) {
			// Found a season directory, return the accumulated path
			if len(directoryAcc) == 0 {
				return "", fmt.Errorf("indeterminable")
			}
			// Reconstruct the path, preserving the structure
			// If directoryAcc starts with "", it's an absolute path
			return strings.Join(directoryAcc, "/"), nil
		}
		directoryAcc = append(directoryAcc, part)
	}

	return "", fmt.Errorf("indeterminable")
}

// MetadataFileHelpersSeasonAndEpisodeFromMediaFilepath/1
func MetadataFileHelpersSeasonAndEpisodeFromMediaFilepath(mediaFilepath string) (string, string, error) {
	// matches s + 1 or more digits + e + 1 or more digits (case-insensitive)
	seasonEpisodeRegex := regexp.MustCompile(`(?i)s(\d+)e(\d+)`)

	matches := seasonEpisodeRegex.FindStringSubmatch(mediaFilepath)
	if len(matches) < 3 {
		return "", "", fmt.Errorf("indeterminable")
	}

	return matches[1], matches[2], nil
}

// metadataFileHelpersGenerateFilepathFor/2
func metadataFileHelpersGenerateFilepathFor(ctx context.Context, a *App, databaseRecord any, filename string) (string, error) {
	metadataDir, err := a.MetadataFileHelpersMetadataDirectoryFor(ctx, databaseRecord)
	if err != nil {
		return "", err
	}

	return filepath.Join(metadataDir, filename), nil
}
