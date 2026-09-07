//go:build windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const ownerSecurityInformation = 0x00000001

var (
	advapi32                       = syscall.NewLazyDLL("advapi32.dll")
	getFileSecurityProc            = advapi32.NewProc("GetFileSecurityW")
	getSecurityDescriptorOwnerProc = advapi32.NewProc("GetSecurityDescriptorOwner")
	equalSidProc                   = advapi32.NewProc("EqualSid")
)

// lockDirectoryOwnedByCurrentUser verifies the Windows security descriptor
// owner against the current process token. os.FileInfo does not carry a
// Windows owner, so the security descriptor API is used instead.
func lockDirectoryOwnedByCurrentUser(path string, _ os.FileInfo) (bool, error) {
	return windowsPathOwnedByCurrentUser(path)
}

func windowsPathOwnedByCurrentUser(path string) (bool, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}

	var needed uint32
	r1, _, firstErr := getFileSecurityProc.Call(
		uintptr(unsafe.Pointer(name)),
		ownerSecurityInformation,
		0,
		0,
		uintptr(unsafe.Pointer(&needed)),
	)
	if r1 != 0 {
		return false, fmt.Errorf("unexpected security descriptor query result")
	}
	if needed == 0 {
		return false, firstErr
	}

	descriptor := make([]byte, needed)
	r1, _, err = getFileSecurityProc.Call(
		uintptr(unsafe.Pointer(name)),
		ownerSecurityInformation,
		uintptr(unsafe.Pointer(&descriptor[0])),
		uintptr(len(descriptor)),
		uintptr(unsafe.Pointer(&needed)),
	)
	if r1 == 0 {
		return false, err
	}

	var owner *syscall.SID
	var ownerDefaulted int32
	r1, _, err = getSecurityDescriptorOwnerProc.Call(
		uintptr(unsafe.Pointer(&descriptor[0])),
		uintptr(unsafe.Pointer(&owner)),
		uintptr(unsafe.Pointer(&ownerDefaulted)),
	)
	if r1 == 0 {
		return false, err
	}
	if owner == nil {
		return false, fmt.Errorf("security descriptor has no owner")
	}

	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return false, err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return false, err
	}

	equal, _, _ := equalSidProc.Call(
		uintptr(unsafe.Pointer(owner)),
		uintptr(unsafe.Pointer(user.User.Sid)),
	)
	return equal != 0, nil
}
