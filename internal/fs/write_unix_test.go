//go:build unix

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

func TestWriteFileUpdatesWriteOnlyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "write-only.env")
	if err := os.WriteFile(path, []byte("old"), 0o200); err != nil {
		t.Fatalf("write initial file failed: %v", err)
	}
	if err := os.Chmod(path, 0o200); err != nil {
		t.Fatalf("set write-only permissions failed: %v", err)
	}

	if err := fs.WriteFile(path, []byte("new")); err != nil {
		t.Fatalf("write write-only file failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file failed: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o200); got != want {
		t.Fatalf("file mode = %o, want %o", got, want)
	}
}
