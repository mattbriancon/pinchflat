package fsutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mattbriancon/pinchflat/internal/fsutil"
)

func TestExistsAndNonEmpty(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name    string
		content []byte
		exists  bool
		want    bool
	}{
		{"has contents", []byte("{}"), true, true},
		{"doesn't exist", nil, false, false},
		{"is empty", []byte(""), true, false},
		{"is whitespace-only", []byte("  \n\n  \r\n  "), true, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(dir, tc.name+".txt")
			if tc.exists {
				if err := os.WriteFile(path, tc.content, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if got := fsutil.ExistsAndNonEmpty(path); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSameFile(t *testing.T) {
	t.Run("returns true if the files are the same", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "test-*.txt")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpfile.Name())
		tmpfile.Close()

		if !fsutil.SameFile(tmpfile.Name(), tmpfile.Name()) {
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
		longPath := filepath.Join("/tmp", "..", shortPath)
		if !fsutil.SameFile(shortPath, longPath) {
			t.Error("expected true for different paths to same file")
		}
	})

	t.Run("returns true if the files are symlinked", func(t *testing.T) {
		dir := t.TempDir()
		sourceFile := filepath.Join(dir, "source.json")
		if err := os.WriteFile(sourceFile, nil, 0644); err != nil {
			t.Fatal(err)
		}
		symlinkPath := filepath.Join(dir, "symlink.json")
		if err := os.Symlink(sourceFile, symlinkPath); err != nil {
			t.Skip("symlink creation failed, skipping test")
		}

		if !fsutil.SameFile(sourceFile, symlinkPath) {
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

		if fsutil.SameFile(tmpfile1.Name(), tmpfile2.Name()) {
			t.Error("expected false for different files")
		}
	})
}

func TestGenerateTmpfile(t *testing.T) {
	t.Run("creates a tmpfile and returns its path", func(t *testing.T) {
		dir := t.TempDir()
		res, err := fsutil.GenerateTmpfile(dir, "json")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(res, ".json") {
			t.Errorf("expected path to end with .json, got %q", res)
		}
		if _, err := os.Stat(res); err != nil {
			t.Error("file should exist")
		}
	})
}

func TestWriteFileAll(t *testing.T) {
	cases := []struct {
		name string
		rel  string
	}{
		{"writes content to a file", "test.json"},
		{"creates directories as needed", filepath.Join("foo", "bar", "file.json")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tc.rel)
			content := "{}"

			if err := fsutil.WriteFileAll(path, content); err != nil {
				t.Fatal(err)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != content {
				t.Errorf("expected %q, got %q", content, string(data))
			}
		})
	}
}

func TestDeleteFileAndRemoveEmptyDirs(t *testing.T) {
	t.Run("deletes file at the provided filepath", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.json")
		os.WriteFile(path, []byte(""), 0644)

		if err := fsutil.DeleteFileAndRemoveEmptyDirs(path); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path); err == nil {
			t.Error("file should not exist")
		}
	})

	t.Run("deletes empty directories", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "foo", "bar", "baz", "qux.json")
		fsutil.WriteFileAll(path, "")

		if err := fsutil.DeleteFileAndRemoveEmptyDirs(path); err != nil {
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
		dir := t.TempDir()
		path1 := filepath.Join(dir, "foo", "bar", "baz", "qux.json")
		path2 := filepath.Join(dir, "foo", "baz.json")
		fsutil.WriteFileAll(path1, "")
		fsutil.WriteFileAll(path2, "")

		if err := fsutil.DeleteFileAndRemoveEmptyDirs(path1); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(path1); err == nil {
			t.Error("file should not exist")
		}
		if _, err := os.Stat(path2); err != nil {
			t.Error("file should still exist")
		}
	})

	t.Run("returns an error if file could not be deleted", func(t *testing.T) {
		if err := fsutil.DeleteFileAndRemoveEmptyDirs("/nonexistent/file.json"); err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

func TestCopyFile(t *testing.T) {
	cases := []struct {
		name string
		dest string
	}{
		{"copies a file from source to destination", "destination.json"},
		{"creates directories as needed", filepath.Join("foo", "bar", "destination.json")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source.json")
			fsutil.WriteFileAll(source, "TEST")
			destination := filepath.Join(dir, tc.dest)

			if err := fsutil.CopyFile(source, destination); err != nil {
				t.Fatal(err)
			}

			data, err := os.ReadFile(destination)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != "TEST" {
				t.Errorf("expected 'TEST', got %q", string(data))
			}
		})
	}
}
