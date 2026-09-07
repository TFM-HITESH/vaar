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

func openTargetFile(path string, flags int) (*os.File, error) {
	access := uint32(syscall.GENERIC_READ)
	switch flags & (os.O_WRONLY | os.O_RDWR) {
	case os.O_WRONLY:
		access = syscall.GENERIC_WRITE
	case os.O_RDWR:
		access = syscall.GENERIC_READ | syscall.GENERIC_WRITE
	}

	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	handle, err := syscall.CreateFile(
		name,
		access,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_NORMAL,
		0,
	)
	if err != nil {
		return nil, err
	}

	file := os.NewFile(uintptr(handle), path)
	if file == nil {
		_ = syscall.CloseHandle(handle)
		return nil, os.ErrInvalid
	}
	return file, nil
}

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
