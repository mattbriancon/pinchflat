package core

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// FilesystemUtilsExistsAndNonempty(ctx, filepath)
// Checks if a file exists and has non-whitespace contents.
func FilesystemUtilsExistsAndNonempty(ctx context.Context, filepath string) bool {
	contents, err := os.ReadFile(filepath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(contents)) != ""
}

// FilesystemUtilsFilepathsReferenceSameFile(ctx, filepath_1, filepath_2)
// Checks if two filepaths reference the same file by comparing inode information.
func FilesystemUtilsFilepathsReferenceSameFile(ctx context.Context, filepath1 string, filepath2 string) bool {
	stat1, err := os.Stat(filepath1)
	if err != nil {
		return false
	}
	stat2, err := os.Stat(filepath2)
	if err != nil {
		return false
	}

	// On Unix: compare device and inode
	// On Windows: compare by os.SameFile (which uses file ID)
	return os.SameFile(stat1, stat2)
}

// (a *App) FilesystemUtilsGenerateMetadataTmpfile(ctx, type)
// Generates a temporary file and returns its path. The file is empty and has the given type.
// Generates all the directories in the path if they don't exist.
func (a *App) FilesystemUtilsGenerateMetadataTmpfile(ctx context.Context, typeStr string) (string, error) {
	filename := StringUtilsRandomString(64)
	firstTwo := filename[0:2]
	secondTwo := filename[2:4]

	filepath := filepath.Join(
		a.Config.TmpfileDirectory,
		firstTwo,
		secondTwo,
		fmt.Sprintf("%s.%s", filename, typeStr),
	)

	if err := FilesystemUtilsWriteP(ctx, filepath, "", []string{}); err != nil {
		return "", err
	}

	return filepath, nil
}

// FilesystemUtilsWriteP(ctx, file, content, modes...)
// Writes content to a file, creating directories as needed.
func FilesystemUtilsWriteP(ctx context.Context, file string, content string, modes []string) error {
	dirname := filepath.Dir(file)

	if err := os.MkdirAll(dirname, 0755); err != nil {
		return err
	}

	// Modes in Elixir are file write options like [:binary, :write, :create]
	// In Go, we just use os.WriteFile which writes with default permissions
	return os.WriteFile(file, []byte(content), 0644)
}

// FilesystemUtilsCpP(ctx, source, destination)
// Copies a file from source to destination, creating directories as needed.
func FilesystemUtilsCpP(ctx context.Context, source string, destination string) error {
	destDir := filepath.Dir(destination)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// (a *App) FilesystemUtilsComputeAndSaveMediaFilesize(ctx, media_item)
// Fetches the file size of a media item and saves it to the database.
func (a *App) FilesystemUtilsComputeAndSaveMediaFilesize(ctx context.Context, mediaItem *MediaItem) (*MediaItem, error) {
	if mediaItem.MediaFilepath == nil {
		return nil, fmt.Errorf("media_filepath is nil")
	}

	stat, err := os.Stat(*mediaItem.MediaFilepath)
	if err != nil {
		return nil, err
	}

	size := stat.Size()
	return a.MediaUpdateMediaItem(ctx, mediaItem, Attrs{
		"media_size_bytes": size,
	})
}

// FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ctx, filepath)
// Deletes a file and removes any empty directories in the path.
// Does NOT remove any directories that are not empty.
func FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ctx context.Context, path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	_ = filesystemUtilsRecursivelyDeleteEmptyDirectories(ctx, dir)

	return nil
}

// filesystemUtilsRecursivelyDeleteEmptyDirectories attempts to delete empty directories.
// Returns nil on any error (directory not empty or doesn't exist), always returns nil.
func filesystemUtilsRecursivelyDeleteEmptyDirectories(ctx context.Context, directory string) error {
	if err := os.Remove(directory); err != nil {
		// Directory not empty or doesn't exist; stop recursion
		return nil
	}

	// Successfully deleted this directory; try parent
	parentDir := filepath.Dir(directory)
	_ = filesystemUtilsRecursivelyDeleteEmptyDirectories(ctx, parentDir)

	return nil
}
