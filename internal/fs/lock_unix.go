//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

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
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
}

func unlockFile(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
