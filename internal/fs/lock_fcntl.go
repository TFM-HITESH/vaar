//go:build aix || (solaris && !illumos)

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"os"
	"syscall"
)

func lockFile(file *os.File) error {
	return syscall.FcntlFlock(file.Fd(), syscall.F_SETLKW, &syscall.Flock_t{
		Type: syscall.F_WRLCK,
	})
}

func unlockFile(file *os.File) error {
	return syscall.FcntlFlock(file.Fd(), syscall.F_SETLK, &syscall.Flock_t{
		Type: syscall.F_UNLCK,
	})
}
