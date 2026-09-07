//go:build aix || (solaris && !illumos)

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

func TestFcntlLocksSerializeGoroutinesInOneProcess(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lock-target")
	if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
		t.Fatalf("create lock target failed: %v", err)
	}

	first, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open first descriptor failed: %v", err)
	}
	defer first.Close()
	second, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open second descriptor failed: %v", err)
	}
	defer second.Close()

	if err := lockFile(first); err != nil {
		t.Fatalf("lock first descriptor failed: %v", err)
	}

	started := make(chan struct{})
	acquired := make(chan error, 1)
	go func() {
		close(started)
		if err := lockFile(second); err != nil {
			acquired <- err
			return
		}
		acquired <- unlockFile(second)
	}()
	<-started

	select {
	case err := <-acquired:
		t.Fatalf("second descriptor acquired process lock early: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	if err := unlockFile(first); err != nil {
		t.Fatalf("unlock first descriptor failed: %v", err)
	}

	select {
	case err := <-acquired:
		if err != nil {
			t.Fatalf("second descriptor failed after release: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("second descriptor did not acquire process lock after release")
	}
}

func TestOpenTargetLockUsesIdentityLockForReadOnlyDestination(t *testing.T) {
	path := filepath.Join(t.TempDir(), "read-only-target")
	if err := os.WriteFile(path, []byte("content"), 0o444); err != nil {
		t.Fatalf("create read-only target failed: %v", err)
	}

	probe, err := os.OpenFile(path, os.O_RDWR, 0)
	if err == nil {
		_ = probe.Close()
		t.Skip("test environment can open read-only files for writing")
	}

	target, targetLock, err := openTargetLock(path)
	if err != nil {
		t.Fatalf("open target lock failed: %v", err)
	}
	if target != nil {
		_ = target.Close()
		t.Fatal("read-only destination returned a writable target descriptor")
	}
	if targetLock == nil {
		t.Fatal("read-only destination did not receive an identity lock")
	}
	if err := unlockAndClose(targetLock); err != nil {
		t.Fatalf("release identity lock failed: %v", err)
	}
}
