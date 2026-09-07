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

// pathLock coordinates Vaar writers for one canonical destination. The lock
// file lives in the operating system's temporary directory and is deliberately
// retained after release: removing it while another process is waiting could
// let a new process create a different inode and bypass the existing lock.
type pathLock struct {
	file *os.File
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
	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open path lock %q: %w", path, err)
	}

	if err := lockFile(file); err != nil {
		return nil, errors.Join(fmt.Errorf("lock path %q: %w", path, err), file.Close())
	}
	return &pathLock{file: file}, nil
}

func (l *pathLock) release() error {
	if l == nil || l.file == nil {
		return nil
	}

	unlockErr := unlockFile(l.file)
	closeErr := l.file.Close()
	l.file = nil
	return errors.Join(unlockErr, closeErr)
}
