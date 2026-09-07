//go:build aix || (solaris && !illumos)

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

func openIdentityTargetLock(path string) (*os.File, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	key, err := fcntlInfoKey(info)
	if err != nil {
		return nil, err
	}

	lockDirectory, err := pathLockDirectory()
	if err != nil {
		return nil, fmt.Errorf("resolve target lock directory: %w", err)
	}
	if err := ensurePrivateLockDir(lockDirectory); err != nil {
		return nil, fmt.Errorf("prepare target lock directory %q: %w", lockDirectory, err)
	}

	digest := sha256.Sum256([]byte(key))
	lockPath := filepath.Join(lockDirectory, "vaar-target-"+hex.EncodeToString(digest[:])+".lock")
	lock, err := openPathLock(lockPath)
	if err != nil {
		return nil, err
	}
	if err := lockFile(lock); err != nil {
		return nil, errors.Join(err, lock.Close())
	}

	current, err := os.Stat(path)
	if err != nil {
		return nil, errors.Join(err, unlockAndClose(lock))
	}
	if !os.SameFile(info, current) {
		return nil, errors.Join(fmt.Errorf("target changed while acquiring identity lock"), unlockAndClose(lock))
	}
	return lock, nil
}
