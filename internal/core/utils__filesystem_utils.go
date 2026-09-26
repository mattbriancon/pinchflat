package core

import (
	"context"
)

// FilesystemUtilsExistsAndNonempty(ctx, filepath)
// Checks if a file exists and has non-whitespace contents.
func FilesystemUtilsExistsAndNonempty(ctx context.Context, filepath string) bool {
	panic("unported: Pinchflat.Utils.FilesystemUtils.exists_and_nonempty?/1")
}

// FilesystemUtilsFilepathsReferenceSameFile(ctx, filepath_1, filepath_2)
// Checks if two filepaths reference the same file.
func FilesystemUtilsFilepathsReferenceSameFile(ctx context.Context, filepath1 string, filepath2 string) bool {
	panic("unported: Pinchflat.Utils.FilesystemUtils.filepaths_reference_same_file?/2")
}

// (a *App) FilesystemUtilsGenerateMetadataTmpfile(ctx, type)
// Generates a temporary file and returns its path. The file is empty and has the given type.
// Generates all the directories in the path if they don't exist.
func (a *App) FilesystemUtilsGenerateMetadataTmpfile(ctx context.Context, typeStr string) (string, error) {
	panic("unported: Pinchflat.Utils.FilesystemUtils.generate_metadata_tmpfile/1")
}

// FilesystemUtilsWriteP(ctx, file, content, modes...)
// Writes content to a file, creating directories as needed.
func FilesystemUtilsWriteP(ctx context.Context, file string, content string, modes []string) error {
	panic("unported: Pinchflat.Utils.FilesystemUtils.write_p/3")
}

// FilesystemUtilsCpP(ctx, source, destination)
// Copies a file from source to destination, creating directories as needed.
func FilesystemUtilsCpP(ctx context.Context, source string, destination string) error {
	panic("unported: Pinchflat.Utils.FilesystemUtils.cp_p!/2")
}

// (a *App) FilesystemUtilsComputeAndSaveMediaFilesize(ctx, media_item)
// Fetches the file size of a media item and saves it to the database.
func (a *App) FilesystemUtilsComputeAndSaveMediaFilesize(ctx context.Context, mediaItem *MediaItem) (*MediaItem, error) {
	panic("unported: Pinchflat.Utils.FilesystemUtils.compute_and_save_media_filesize/1")
}

// FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ctx, filepath)
// Deletes a file and removes any empty directories in the path.
// Does NOT remove any directories that are not empty.
func FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ctx context.Context, filepath string) error {
	panic("unported: Pinchflat.Utils.FilesystemUtils.delete_file_and_remove_empty_directories/1")
}
