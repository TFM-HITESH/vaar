//go:build darwin || windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"path/filepath"
	"testing"
	"time"
)

func TestAcquirePathLockSerializesCaseVariantsForMissingDestination(t *testing.T) {
	root := t.TempDir()
	first, err := acquirePathLock(filepath.Join(root, ".env"))
	if err != nil {
		t.Fatalf("acquire first path lock failed: %v", err)
	}

	result := make(chan error, 1)
	go func() {
		second, err := acquirePathLock(filepath.Join(root, ".ENV"))
		if err == nil {
			err = second.release()
		}
		result <- err
	}()

	select {
	case err := <-result:
		t.Fatalf("case-variant path lock acquired early: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	if err := first.release(); err != nil {
		t.Fatalf("release first path lock failed: %v", err)
	}

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("case-variant path lock failed after release: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("case-variant path lock did not acquire after release")
	}
}
