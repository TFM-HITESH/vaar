/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewAtomicFileSerializesPathWriters(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(root, ".env")
	if err := os.WriteFile(destination, []byte("KEY=value\n"), 0o644); err != nil {
		t.Fatalf("write destination failed: %v", err)
	}

	atomic, err := NewAtomicFile(destination)
	if err != nil {
		t.Fatalf("create atomic file failed: %v", err)
	}

	started := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		close(started)
		result <- WriteFile(destination, []byte("KEY=changed\n"))
	}()
	<-started

	select {
	case err := <-result:
		t.Fatalf("second writer acquired the path lock early: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	if err := atomic.Cleanup(); err != nil {
		t.Fatalf("cleanup atomic file failed: %v", err)
	}

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("second writer failed after release: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("second writer did not acquire the path lock after release")
	}

	data, err := os.ReadFile(destination)
	if err != nil {
		t.Fatalf("read destination failed: %v", err)
	}
	if string(data) != "KEY=changed\n" {
		t.Fatalf("destination = %q, want changed content", data)
	}
}
