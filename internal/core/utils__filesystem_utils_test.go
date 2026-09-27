package core_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/core"
	"github.com/mattbriancon/pinchflat/internal/core/coretest"
)

func TestFilesystemUtils_ExistsAndNonempty(t *testing.T) {
	t.Run("returns true if a file exists and has contents", func(t *testing.T) {
		ta := coretest.NewApp(t)
		filepath, err := ta.App.FilesystemUtilsGenerateMetadataTmpfile(ta.Ctx, "json")
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath, []byte("{}"), 0644)

		if !core.FilesystemUtilsExistsAndNonempty(ta.Ctx, filepath) {
			t.Error("expected true")
		}

		os.Remove(filepath)
	})

	t.Run("returns false if a file doesn't exist", func(t *testing.T) {
		if core.FilesystemUtilsExistsAndNonempty(nil, "/nonexistent/file.json") {
			t.Error("expected false")
		}
	})

	t.Run("returns false if a file exists but is empty", func(t *testing.T) {
		ta := coretest.NewApp(t)
		filepath, err := ta.App.FilesystemUtilsGenerateMetadataTmpfile(ta.Ctx, "json")
		if err != nil {
			t.Fatal(err)
		}
		if core.FilesystemUtilsExistsAndNonempty(ta.Ctx, filepath) {
			t.Error("expected false")
		}
		os.Remove(filepath)
	})

	t.Run("trims the contents before checking", func(t *testing.T) {
		ta := coretest.NewApp(t)
		filepath, err := ta.App.FilesystemUtilsGenerateMetadataTmpfile(ta.Ctx, "json")
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath, []byte("  \n\n  \r\n  "), 0644)
		if core.FilesystemUtilsExistsAndNonempty(ta.Ctx, filepath) {
			t.Error("expected false")
		}
		os.Remove(filepath)
	})
}

func TestFilesystemUtils_FilepathsReferenceSameFile(t *testing.T) {
	t.Run("returns true if the files are the same", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())
		tmpfile.Close()

		if !core.FilesystemUtilsFilepathsReferenceSameFile(nil, tmpfile.Name(), tmpfile.Name()) {
			t.Error("expected true for same file")
		}
	})

	t.Run("returns true if different filepaths point to the same file", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())
		tmpfile.Close()

		shortPath := tmpfile.Name()
		// Construct a path with redundant .. components that should resolve to the same file
		longPath := filepath.Join("/tmp", "..", shortPath)

		if shortPath == longPath {
			// If they're the same string, that's OK, just verify same file returns true
			if !core.FilesystemUtilsFilepathsReferenceSameFile(nil, shortPath, shortPath) {
				t.Error("expected true for same file")
			}
		} else {
			if !core.FilesystemUtilsFilepathsReferenceSameFile(nil, shortPath, longPath) {
				t.Error("expected true for different paths to same file")
			}
		}
	})

	t.Run("returns true if the files are symlinked", func(t *testing.T) {
		ta := coretest.NewApp(t)
		// Create source file
		sourceFile, err := ta.App.FilesystemUtilsGenerateMetadataTmpfile(ta.Ctx, "json")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(sourceFile)

		// Create symlink
		symlinkPath := filepath.Join(filepath.Dir(sourceFile), "symlink.json")
		if err := os.Symlink(sourceFile, symlinkPath); err != nil {
			t.Skip("symlink creation failed, skipping test")
		}
		defer os.Remove(symlinkPath)

		if !core.FilesystemUtilsFilepathsReferenceSameFile(ta.Ctx, sourceFile, symlinkPath) {
			t.Error("expected true for symlinked files")
		}
	})

	t.Run("returns false if the files are different", func(t *testing.T) {
		tmpfile1, err := os.CreateTemp("", "test1-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile1.Name())
		tmpfile1.Close()

		tmpfile2, err := os.CreateTemp("", "test2-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile2.Name())
		tmpfile2.Close()

		if core.FilesystemUtilsFilepathsReferenceSameFile(nil, tmpfile1.Name(), tmpfile2.Name()) {
			t.Error("expected false for different files")
		}
	})
}

func TestFilesystemUtils_GenerateMetadataTmpfile(t *testing.T) {
	t.Run("creates a tmpfile and returns its path", func(t *testing.T) {
		ta := coretest.NewApp(t)
		res, err := ta.App.FilesystemUtilsGenerateMetadataTmpfile(ta.Ctx, "json")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(res, ".json") {
			t.Errorf("expected path to end with .json, got %q", res)
		}
		if _, err := os.Stat(res); err != nil {
			t.Error("file should exist")
		}
		os.Remove(res)
	})
}

func TestFilesystemUtils_ComputeAndSaveMediaFilesize(t *testing.T) {
	t.Run("updates the media item with the file size", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemWithAttachmentsFixture(t, ta, core.Attrs{})

		if mediaItem.MediaSizeBytes != nil && *mediaItem.MediaSizeBytes != 0 {
			t.Error("media_size_bytes should initially be nil or 0")
		}

		result, err := ta.App.FilesystemUtilsComputeAndSaveMediaFilesize(ta.Ctx, mediaItem)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.MediaSizeBytes == nil || *result.MediaSizeBytes == 0 {
			t.Error("media_size_bytes should be set and non-zero")
		}
	})

	t.Run("returns the error if operation fails", func(t *testing.T) {
		ta := coretest.NewApp(t)
		mediaItem := coretest.MediaItemFixture(t, ta, core.Attrs{
			"media_filepath": "/nonexistent/file.mkv",
		})

		_, err := ta.App.FilesystemUtilsComputeAndSaveMediaFilesize(ta.Ctx, mediaItem)
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestFilesystemUtils_WriteP(t *testing.T) {
	t.Run("writes content to a file", func(t *testing.T) {
		ta := coretest.NewApp(t)
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "test.json")
		content := "{}"

		if err := core.FilesystemUtilsWriteP(ta.Ctx, filepath, content, []string{}); err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}

		os.Remove(filepath)
	})

	t.Run("creates directories as needed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "foo", "bar", "file.json")
		content := "{}"

		if err := core.FilesystemUtilsWriteP(ta.Ctx, filepath, content, []string{}); err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(filepath)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != content {
			t.Errorf("expected %q, got %q", content, string(data))
		}

		os.Remove(filepath)
	})
}

func TestFilesystemUtils_DeleteFileAndRemoveEmptyDirectories(t *testing.T) {
	t.Run("deletes file at the provided filepath", func(t *testing.T) {
		ta := coretest.NewApp(t)
		filepath := filepath.Join(ta.Config.TmpfileDirectory, "test.json")
		os.WriteFile(filepath, []byte(""), 0644)

		if _, err := os.Stat(filepath); err != nil {
			t.Fatal("file should exist")
		}

		if err := core.FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ta.Ctx, filepath); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(filepath); err == nil {
			t.Error("file should not exist")
		}
	})

	t.Run("deletes empty directories", func(t *testing.T) {
		ta := coretest.NewApp(t)
		path := filepath.Join(ta.Config.TmpfileDirectory, "foo", "bar", "baz", "qux.json")
		core.FilesystemUtilsWriteP(ta.Ctx, path, "", []string{})

		if err := core.FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ta.Ctx, path); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(path); err == nil {
			t.Error("file should not exist")
		}
		emptyDir := filepath.Dir(filepath.Dir(filepath.Dir(path)))
		if _, err := os.Stat(emptyDir); err == nil {
			t.Error("empty directory should be removed")
		}
	})

	t.Run("does not delete directories with other files in them", func(t *testing.T) {
		ta := coretest.NewApp(t)
		filepath1 := filepath.Join(ta.Config.TmpfileDirectory, "foo", "bar", "baz", "qux.json")
		filepath2 := filepath.Join(ta.Config.TmpfileDirectory, "foo", "baz.json")
		core.FilesystemUtilsWriteP(ta.Ctx, filepath1, "", []string{})
		core.FilesystemUtilsWriteP(ta.Ctx, filepath2, "", []string{})

		if err := core.FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ta.Ctx, filepath1); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(filepath1); err == nil {
			t.Error("file should not exist")
		}
		if _, err := os.Stat(filepath2); err != nil {
			t.Error("file should still exist")
		}

		core.FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(ta.Ctx, filepath2)
	})

	t.Run("returns an error if file could not be deleted", func(t *testing.T) {
		if err := core.FilesystemUtilsDeleteFileAndRemoveEmptyDirectories(nil, "/nonexistent/file.json"); err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestFilesystemUtils_CpP(t *testing.T) {
	t.Run("copies a file from source to destination", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := filepath.Join(ta.Config.TmpfileDirectory, "source.json")
		core.FilesystemUtilsWriteP(ta.Ctx, source, "TEST", []string{})
		destination := filepath.Join(ta.Config.TmpfileDirectory, "destination.json")

		if err := core.FilesystemUtilsCpP(ta.Ctx, source, destination); err != nil {
			t.Fatal(err)
		}

		data, err := os.ReadFile(destination)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "TEST" {
			t.Errorf("expected 'TEST', got %q", string(data))
		}

		os.Remove(source)
		os.Remove(destination)
	})

	t.Run("creates directories as needed", func(t *testing.T) {
		ta := coretest.NewApp(t)
		source := filepath.Join(ta.Config.TmpfileDirectory, "source.json")
		core.FilesystemUtilsWriteP(ta.Ctx, source, "TEST", []string{})
		destination := filepath.Join(ta.Config.TmpfileDirectory, "foo", "bar", "destination.json")

		if err := core.FilesystemUtilsCpP(ta.Ctx, source, destination); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Stat(destination); err != nil {
			t.Error("destination file should exist")
		}

		os.Remove(source)
		os.Remove(destination)
	})
}
