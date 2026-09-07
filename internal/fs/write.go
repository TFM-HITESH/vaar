/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"errors"
	"io"
	"os"
)

const defaultFileMode os.FileMode = 0o644

// WriteFile writes data to path, preserving the existing permission bits when
// path already identifies a file. It coordinates with AtomicFile writers for
// the same pathname and currently resolved target. New files use the standard
// 0644 mode, subject to the process umask.
func WriteFile(path string, data []byte) error {
	lock, err := acquirePathLock(path)
	if err != nil {
		return err
	}

	var writeErr error
	if lock.targetFile != nil {
		writeErr = writeLockedTarget(lock.targetFile, data)
	} else {
		perm := defaultFileMode
		if info, err := os.Stat(path); err == nil {
			perm = info.Mode().Perm()
		}
		writeErr = os.WriteFile(path, data, perm)
	}

	return errors.Join(writeErr, lock.release())
}

func writeLockedTarget(file *os.File, data []byte) error {
	if _, err := file.Seek(0, 0); err != nil {
		return err
	}
	if err := file.Truncate(0); err != nil {
		return err
	}
	written, err := file.Write(data)
	if err != nil {
		return err
	}
	if written != len(data) {
		return io.ErrShortWrite
	}
	return nil
}
