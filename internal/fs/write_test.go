/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/envaar/vaar/internal/fs"
)

func TestWriteFilePreservesExistingPermissions(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "config.env")

	if err := os.WriteFile(path, []byte("old"), 0o601); err != nil {
		t.Fatalf("write initial file failed: %v", err)
	}
	initialInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat initial file failed: %v", err)
	}
	initialPerm := initialInfo.Mode().Perm()

	if err := fs.WriteFile(path, []byte("new")); err != nil {
		t.Fatalf("write existing file failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat rewritten file failed: %v", err)
	}
	if got, want := info.Mode().Perm(), initialPerm; got != want {
		t.Fatalf("rewritten file permissions changed: got %o want %o", got, want)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten file failed: %v", err)
	}
	if got, want := string(contents), "new"; got != want {
		t.Fatalf("rewritten file contents changed unexpectedly: got %q want %q", got, want)
	}
}

func TestWriteFileUsesDefaultModeForNewFiles(t *testing.T) {
	root := t.TempDir()
	reference := filepath.Join(root, "reference")
	path := filepath.Join(root, "created")

	if err := os.WriteFile(reference, []byte("reference"), 0o644); err != nil {
		t.Fatalf("write reference file failed: %v", err)
	}
	if err := fs.WriteFile(path, []byte("created")); err != nil {
		t.Fatalf("write new file failed: %v", err)
	}

	referenceInfo, err := os.Stat(reference)
	if err != nil {
		t.Fatalf("stat reference file failed: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat created file failed: %v", err)
	}
	if got, want := info.Mode().Perm(), referenceInfo.Mode().Perm(); got != want {
		t.Fatalf("new file permissions differ from default: got %o want %o", got, want)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read created file failed: %v", err)
	}
	if got, want := string(contents), "created"; got != want {
		t.Fatalf("created file contents differ: got %q want %q", got, want)
	}
}
