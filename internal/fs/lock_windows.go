//go:build windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"os"
	"syscall"
	"unsafe"
)

const lockFileExclusive = 0x00000002

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	lockFileExProc = kernel32.NewProc("LockFileEx")
	unlockFileProc = kernel32.NewProc("UnlockFileEx")
)

func lockFile(file *os.File) error {
	var overlapped syscall.Overlapped
	r1, _, err := lockFileExProc.Call(
		file.Fd(),
		lockFileExclusive,
		0,
		uintptr(^uint32(0)),
		uintptr(^uint32(0)),
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func unlockFile(file *os.File) error {
	var overlapped syscall.Overlapped
	r1, _, err := unlockFileProc.Call(
		file.Fd(),
		0,
		uintptr(^uint32(0)),
		uintptr(^uint32(0)),
		uintptr(unsafe.Pointer(&overlapped)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}
