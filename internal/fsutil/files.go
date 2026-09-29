// Package fsutil holds small, dependency-free filesystem helpers.
package fsutil

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExistsAndNonEmpty reports whether path exists and has non-whitespace
// contents.
func ExistsAndNonEmpty(path string) bool {
	contents, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(contents)) != ""
}

// SameFile reports whether path1 and path2 refer to the same file on disk
// (comparing device and inode, or symlink target).
func SameFile(path1, path2 string) bool {
	stat1, err := os.Stat(path1)
	if err != nil {
		return false
	}
	stat2, err := os.Stat(path2)
	if err != nil {
		return false
	}
	return os.SameFile(stat1, stat2)
}

// GenerateTmpfile creates an empty file of the given type (extension) under
// dir, in a two-level directory named from a random string, and returns its
// path.
func GenerateTmpfile(dir string, ext string) (string, error) {
	name := randomString(64)
	path := filepath.Join(dir, name[0:2], name[2:4], name+"."+ext)
	if err := WriteFileAll(path, ""); err != nil {
		return "", err
	}
	return path, nil
}

// WriteFileAll writes content to file, creating its parent directories as
// needed.
func WriteFileAll(file string, content string) error {
	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return err
	}
	return os.WriteFile(file, []byte(content), 0644)
}

// CopyFile copies source to destination, creating destination's parent
// directories as needed.
func CopyFile(source, destination string) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
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

// DeleteFileAndRemoveEmptyDirs deletes the file at path, then removes any
// now-empty directories up the tree. It never removes a directory that
// still has other files in it.
func DeleteFileAndRemoveEmptyDirs(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	removeEmptyDirs(filepath.Dir(path))
	return nil
}

// removeEmptyDirs deletes dir and its ancestors while each is empty,
// stopping (silently) at the first one that isn't.
func removeEmptyDirs(dir string) {
	if err := os.Remove(dir); err != nil {
		return
	}
	removeEmptyDirs(filepath.Dir(dir))
}

// randomString returns a random lower-case hex string of the given length.
func randomString(length int) string {
	randomBytes := make([]byte, (length+1)/2)
	if _, err := rand.Read(randomBytes); err != nil {
		return ""
	}

	hexStr := hex.EncodeToString(randomBytes)
	if len(hexStr) > length {
		return hexStr[:length]
	}
	return hexStr
}
