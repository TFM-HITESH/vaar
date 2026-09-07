/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestEnsurePrivateLockDirCreatesRestrictiveDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vaar-locks")

	if err := ensurePrivateLockDir(path); err != nil {
		t.Fatalf("ensure private lock directory failed: %v", err)
	}

	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("stat lock directory failed: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("lock directory is a symlink")
	}
	if !info.IsDir() {
		t.Fatal("lock path is not a directory")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o700 {
		got, want := info.Mode().Perm(), os.FileMode(0o700)
		t.Fatalf("lock directory permissions = %o, want %o", got, want)
	}
}

func TestEnsurePrivateLockDirRejectsPreexistingSymlink(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "vaar-locks")
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatalf("create symlink target failed: %v", err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	if err := ensurePrivateLockDir(path); err == nil {
		t.Fatal("expected preexisting lock-directory symlink to be rejected")
	}
}

func TestOpenPathLockRejectsPreexistingSymlink(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "lock")
	target := filepath.Join(root, "target")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatalf("create symlink target failed: %v", err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	file, err := openPathLock(path)
	if err == nil {
		_ = file.Close()
		t.Fatal("expected preexisting lock-file symlink to be rejected")
	}
}

func TestVerifyTargetLockRejectsReplacedPath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "destination")
	replacement := filepath.Join(root, "replacement")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatalf("create destination failed: %v", err)
	}
	if err := os.WriteFile(replacement, []byte("new"), 0o600); err != nil {
		t.Fatalf("create replacement failed: %v", err)
	}

	target, err := os.Open(path)
	if err != nil {
		t.Fatalf("open target failed: %v", err)
	}
	defer target.Close()
	if err := os.Rename(replacement, path); err != nil {
		t.Fatalf("replace destination failed: %v", err)
	}

	if err := verifyTargetLock(path, target); err == nil {
		t.Fatal("expected replaced path to fail target-lock verification")
	}
}

func TestAcquirePathLockUsesPrivateDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "destination")
	lock, err := acquirePathLock(path)
	if err != nil {
		t.Fatalf("acquire path lock failed: %v", err)
	}
	t.Cleanup(func() {
		if err := lock.release(); err != nil {
			t.Errorf("release path lock failed: %v", err)
		}
	})

	want, err := pathLockDirectory()
	if err != nil {
		t.Fatalf("resolve path lock directory failed: %v", err)
	}
	if got := filepath.Dir(lock.pathFile.Name()); got != want {
		t.Fatalf("path lock directory = %q, want %q", got, want)
	}
}
