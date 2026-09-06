/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/envaar/vaar/internal/fs"
)

func TestAtomicFileReplacesMissingDestination(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "lint.json")
	payload := []byte("replacement")

	file, err := fs.NewAtomicFile(destination)
	if err != nil {
		t.Fatalf("create atomic file failed: %v", err)
	}
	defer func() { _ = file.Cleanup() }()

	if _, err := file.Write(payload); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := file.Finalize(); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	assertFileBytes(t, destination, payload)
	assertNoAtomicTemporaryFiles(t, root)
}

func TestAtomicFileReplacesExistingFile(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "lint.json")
	if err := os.WriteFile(destination, []byte("stale"), 0o644); err != nil {
		t.Fatalf("write stale file failed: %v", err)
	}
	payload := []byte("replacement")

	file, err := fs.NewAtomicFile(destination)
	if err != nil {
		t.Fatalf("create atomic file failed: %v", err)
	}
	defer func() { _ = file.Cleanup() }()

	if _, err := file.Write(payload); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if err := file.Finalize(); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	assertFileBytes(t, destination, payload)
	assertNoAtomicTemporaryFiles(t, root)
}

func TestAtomicFileDoesNotReplaceDirectory(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatalf("create destination directory failed: %v", err)
	}
	kept := filepath.Join(destination, "keep.txt")
	if err := os.WriteFile(kept, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write directory sentinel failed: %v", err)
	}

	_, err := fs.NewAtomicFile(destination)
	if err == nil {
		t.Fatal("expected directory destination to be rejected")
	}
	if !errors.Is(err, fs.ErrIsDirectory) {
		t.Fatalf("expected ErrIsDirectory, got %v", err)
	}

	info, err := os.Stat(destination)
	if err != nil {
		t.Fatalf("stat destination failed: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("destination is no longer a directory")
	}
	assertFileBytes(t, kept, []byte("keep"))
	assertNoAtomicTemporaryFiles(t, root)
}

func TestAtomicFilePreservesDirectoryCreatedAfterOpen(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "destination")

	file, err := fs.NewAtomicFile(destination)
	if err != nil {
		t.Fatalf("create atomic file failed: %v", err)
	}
	defer func() { _ = file.Cleanup() }()
	if _, err := file.Write([]byte("replacement")); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatalf("create replacement directory failed: %v", err)
	}
	kept := filepath.Join(destination, "keep.txt")
	if err := os.WriteFile(kept, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write directory sentinel failed: %v", err)
	}

	err = file.Finalize()
	if err == nil {
		t.Fatal("expected finalize to reject a directory created after open")
	}
	if !errors.Is(err, fs.ErrIsDirectory) {
		t.Fatalf("expected ErrIsDirectory, got %v", err)
	}

	info, err := os.Stat(destination)
	if err != nil {
		t.Fatalf("stat destination failed: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("destination is no longer a directory")
	}
	assertFileBytes(t, kept, []byte("keep"))
	assertNoAtomicTemporaryFiles(t, root)
}

func TestAtomicFileCleanupRemovesTemporaryFile(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "lint.json")

	file, err := fs.NewAtomicFile(destination)
	if err != nil {
		t.Fatalf("create atomic file failed: %v", err)
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read temporary directory failed: %v", err)
	}
	if len(entries) != 1 || !strings.HasPrefix(entries[0].Name(), "vaar-atomic-") {
		t.Fatalf("expected one atomic temporary file, got %v", entries)
	}

	if err := file.Cleanup(); err != nil {
		t.Fatalf("first cleanup failed: %v", err)
	}
	if err := file.Cleanup(); err != nil {
		t.Fatalf("second cleanup should be harmless: %v", err)
	}
	assertNoAtomicTemporaryFiles(t, root)
}

func TestAtomicFileRejectsWritesAfterCleanup(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, "lint.json")

	file, err := fs.NewAtomicFile(destination)
	if err != nil {
		t.Fatalf("create atomic file failed: %v", err)
	}
	if err := file.Cleanup(); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}

	if _, err := file.Write([]byte("replacement")); !errors.Is(err, fs.ErrAtomicFileClosed) {
		t.Fatalf("expected ErrAtomicFileClosed, got %v", err)
	}
}

func TestValidateFileDestinationRejectsTrailingSeparator(t *testing.T) {
	path := filepath.Join(t.TempDir(), "destination") + string(os.PathSeparator)

	if err := fs.ValidateFileDestination(path); !errors.Is(err, fs.ErrIsDirectory) {
		t.Fatalf("expected ErrIsDirectory, got %v", err)
	}
}

func TestTempDirForPathUsesDestinationDirectory(t *testing.T) {
	path := filepath.Join("root", "reports", "lint.json")
	if got, want := fs.TempDirForPath(path), filepath.Dir(path); got != want {
		t.Fatalf("unexpected temporary directory: got %q want %q", got, want)
	}
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q failed: %v", path, err)
	}
	if string(got) != string(want) {
		t.Fatalf("unexpected bytes in %q: got %q want %q", path, got, want)
	}
}

func assertNoAtomicTemporaryFiles(t *testing.T, root string) {
	t.Helper()

	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read directory failed: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "vaar-atomic-") {
			t.Fatalf("temporary atomic file was not cleaned up: %q", entry.Name())
		}
	}
}
