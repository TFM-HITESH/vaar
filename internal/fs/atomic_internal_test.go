/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicFileWriteReportsCleanupFailure(t *testing.T) {
	root := t.TempDir()
	temporary := filepath.Join(root, "temporary")
	if err := os.Mkdir(temporary, 0o755); err != nil {
		t.Fatalf("create temporary directory failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(temporary, "keep"), []byte("keep"), 0o644); err != nil {
		t.Fatalf("write directory sentinel failed: %v", err)
	}

	file, err := os.CreateTemp(root, "closed-")
	if err != nil {
		t.Fatalf("create closed file failed: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("close file failed: %v", err)
	}

	atomic := &AtomicFile{
		temporary: temporary,
		file:      file,
	}
	_, err = atomic.Write([]byte("replacement"))
	if err == nil {
		t.Fatal("expected write failure")
	}
	if !strings.Contains(err.Error(), temporary) {
		t.Fatalf("write error = %v, want cleanup path", err)
	}
	if atomic.temporary != temporary {
		t.Fatalf("temporary path = %q, want retained path after cleanup failure", atomic.temporary)
	}
}
