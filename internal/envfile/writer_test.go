/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package envfile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/envaar/vaar/internal/envfile"
)

func TestWritePreservesExistingPermissions(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("OLD=value\n"), 0o640); err != nil {
		t.Fatalf("write initial file failed: %v", err)
	}
	initialInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat initial file failed: %v", err)
	}
	initialPerm := initialInfo.Mode().Perm()

	if err := envfile.Write(path, []byte("NEW=value\n")); err != nil {
		t.Fatalf("write envfile failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat rewritten file failed: %v", err)
	}
	if got, want := info.Mode().Perm(), initialPerm; got != want {
		t.Fatalf("rewritten envfile permissions changed: got %o want %o", got, want)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read rewritten envfile failed: %v", err)
	}
	if got, want := string(contents), "NEW=value\n"; got != want {
		t.Fatalf("rewritten envfile contents changed unexpectedly: got %q want %q", got, want)
	}
}
