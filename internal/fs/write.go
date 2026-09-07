/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"errors"
	"os"
)

const defaultFileMode os.FileMode = 0o644

// WriteFile writes data to path, preserving the existing permission bits when
// path already identifies a file. It coordinates with AtomicFile writers for
// the same destination. New files use the standard 0644 mode, subject to the
// process umask.
func WriteFile(path string, data []byte) error {
	lock, err := acquirePathLock(path)
	if err != nil {
		return err
	}

	perm := defaultFileMode
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	}

	return errors.Join(os.WriteFile(path, data, perm), lock.release())
}
