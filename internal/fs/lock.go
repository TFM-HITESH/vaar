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
)

// pathLock coordinates Vaar writers for one destination. The retained path
// lock serializes replacement of the same pathname, while the target lock
// serializes aliases that currently resolve to the same filesystem entry.
// This two-level lock is needed because atomic replacement changes the target
// inode behind a pathname.
type pathLock struct {
	pathFile   *os.File
	targetFile *os.File
}

func acquirePathLock(path string) (*pathLock, error) {
	key, err := CanonicalPath(path)
	if err != nil {
		key, err = filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve lock path %q: %w", path, err)
		}
	}

	digest := sha256.Sum256([]byte(key))
	lockPath := filepath.Join(os.TempDir(), "vaar-path-"+hex.EncodeToString(digest[:])+".lock")
	pathFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o666)
	if err != nil {
		return nil, fmt.Errorf("open path lock %q: %w", path, err)
	}
	// The lock file contains no data and is retained so waiting processes never
	// observe an inode split. Explicitly make it reopenable by another
	// legitimate user of a shared checkout, independent of the creator's umask.
	if err := pathFile.Chmod(0o666); err != nil {
		return nil, errors.Join(fmt.Errorf("set path lock permissions %q: %w", path, err), pathFile.Close())
	}

	if err := lockFile(pathFile); err != nil {
		return nil, errors.Join(fmt.Errorf("lock path %q: %w", path, err), pathFile.Close())
	}

	targetFile, err := openTargetLock(path)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("open target lock %q: %w", path, err), unlockAndClose(pathFile))
	}
	return &pathLock{pathFile: pathFile, targetFile: targetFile}, nil
}

func (l *pathLock) release() error {
	if l == nil {
		return nil
	}

	var targetErr error
	if l.targetFile != nil {
		targetErr = unlockAndClose(l.targetFile)
		l.targetFile = nil
	}

	var pathErr error
	if l.pathFile != nil {
		pathErr = unlockAndClose(l.pathFile)
		l.pathFile = nil
	}
	return errors.Join(targetErr, pathErr)
}

func openTargetLock(path string) (*os.File, error) {
	target, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if err := lockFile(target); err != nil {
		return nil, errors.Join(err, target.Close())
	}
	return target, nil
}

func unlockAndClose(file *os.File) error {
	if file == nil {
		return nil
	}
	return errors.Join(unlockFile(file), file.Close())
}
