/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const pathLockDirectoryPrefix = "vaar-locks-"

// pathLock coordinates Vaar writers for one destination. The retained path
// lock serializes replacement of the same pathname within the current user's
// lock namespace, while the target lock serializes aliases that currently
// resolve to the same filesystem entry. This two-level lock is needed because
// atomic replacement changes the target inode behind a pathname.
type pathLock struct {
	pathFile   *os.File
	targetFile *os.File
	targetLock *os.File
}

func acquirePathLock(path string) (*pathLock, error) {
	key, err := CanonicalPath(path)
	if err != nil {
		key, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve lock path %q: %w", path, err)
		}
	}
	key = normalizeLockPath(key)

	lockDirectory, err := pathLockDirectory()
	if err != nil {
		return nil, fmt.Errorf("resolve path lock directory: %w", err)
	}
	if err := ensurePrivateLockDir(lockDirectory); err != nil {
		return nil, fmt.Errorf("prepare path lock directory %q: %w", lockDirectory, err)
	}

	digest := sha256.Sum256([]byte(key))
	lockPath := filepath.Join(lockDirectory, "vaar-path-"+hex.EncodeToString(digest[:])+".lock")
	pathFile, err := openPathLock(lockPath)
	if err != nil {
		return nil, fmt.Errorf("open path lock %q: %w", path, err)
	}

	if err := lockFile(pathFile); err != nil {
		return nil, errors.Join(fmt.Errorf("lock path %q: %w", path, err), pathFile.Close())
	}

	targetFile, targetLock, err := openTargetLock(path)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open target lock %q: %w", path, err), unlockAndClose(pathFile))
	}
	return &pathLock{pathFile: pathFile, targetFile: targetFile, targetLock: targetLock}, nil
}

func pathLockDirectory() (string, error) {
	ownerPath, err := os.UserCacheDir()
	if err != nil {
		ownerPath, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve per-user lock namespace: %w", err)
		}
	}

	digest := sha256.Sum256([]byte(filepath.Clean(ownerPath)))
	return filepath.Join(os.TempDir(), pathLockDirectoryPrefix+hex.EncodeToString(digest[:])), nil
}

func ensurePrivateLockDir(path string) error {
	if err := os.Mkdir(path, 0o700); err != nil && !os.IsExist(err) {
		return err
	}

	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("lock directory is a symlink")
	}
	if !info.IsDir() {
		return fmt.Errorf("lock path is not a directory")
	}
	owned, err := lockDirectoryOwnedByCurrentUser(path, info)
	if err != nil {
		return err
	}
	if !owned {
		return fmt.Errorf("lock directory is not owned by the current user")
	}

	// Windows does not expose POSIX permission bits through os.FileMode. On
	// Unix-like systems, remove group/world access even when a directory from a
	// previous process already exists.
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		if err := os.Chmod(path, 0o700); err != nil {
			return err
		}
		info, err = os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("lock directory changed while securing it")
		}
		if info.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("lock directory is not private")
		}
		owned, err := lockDirectoryOwnedByCurrentUser(path, info)
		if err != nil {
			return err
		}
		if !owned {
			return fmt.Errorf("lock directory ownership changed while securing it")
		}
	}
	return nil
}

func openPathLock(path string) (*os.File, error) {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("lock path is a symlink")
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("lock path is not a regular file")
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// O_EXCL handles the first-creation race. If another process already
	// created the retained lock file, validate it with Lstat before reopening.
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		if !os.IsExist(err) {
			return nil, err
		}

		info, statErr := os.Lstat(path)
		if statErr != nil {
			return nil, statErr
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("lock path is a symlink")
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("lock path is not a regular file")
		}

		file, err = os.OpenFile(path, os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
	}
	if err := file.Chmod(0o600); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file, nil
}

func (l *pathLock) release() error {
	if l == nil {
		return nil
	}

	var targetErr error
	if l.targetLock != nil {
		targetErr = unlockAndClose(l.targetLock)
		l.targetLock = nil
	}
	l.targetFile = nil

	var pathErr error
	if l.pathFile != nil {
		pathErr = unlockAndClose(l.pathFile)
		l.pathFile = nil
	}
	return errors.Join(targetErr, pathErr)
}

func openTargetLock(path string) (targetFile, targetLock *os.File, err error) {
	var lastErr error
	for _, flags := range []int{os.O_RDWR, os.O_WRONLY, os.O_RDONLY} {
		target, err := openTargetFile(path, flags)
		if err == nil {
			if err := lockFile(target); err != nil {
				lastErr = errors.Join(err, target.Close())
				continue
			}
			if err := verifyTargetLock(path, target); err != nil {
				return nil, nil, errors.Join(err, unlockAndClose(target))
			}
			return target, target, nil
		}
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		lastErr = err
	}

	identityLock, identityErr := openIdentityTargetLock(path)
	if identityErr == nil {
		return nil, identityLock, nil
	}
	return nil, nil, errors.Join(lastErr, identityErr)
}

func verifyTargetLock(path string, target *os.File) error {
	current, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat target after locking: %w", err)
	}
	targetInfo, err := target.Stat()
	if err != nil {
		return fmt.Errorf("stat locked target: %w", err)
	}
	if !os.SameFile(current, targetInfo) {
		return fmt.Errorf("target changed while acquiring lock")
	}
	return nil
}

func unlockAndClose(file *os.File) error {
	if file == nil {
		return nil
	}
	return errors.Join(unlockFile(file), file.Close())
}
