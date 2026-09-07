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

const (
	ownerSecurityInformation          = 0x00000001
	daclSecurityInformation           = 0x00000004
	securityInformation               = ownerSecurityInformation | daclSecurityInformation
	aclInformationSize                = 2
	accessAllowedACEType              = 0
	accessDeniedACEType               = 1
	accessDeniedCallbackACEType       = 3
	accessDeniedObjectACEType         = 6
	accessDeniedCallbackObjectACEType = 7
	lockDirectoryWriteMask            = 0x00000002 | // FILE_ADD_FILE
		0x00000004 | // FILE_ADD_SUBDIRECTORY
		0x00000010 | // FILE_WRITE_EA
		0x00000040 | // FILE_DELETE_CHILD
		0x00010000 | // DELETE
		0x00040000 | // WRITE_DAC
		0x00080000 | // WRITE_OWNER
		0x10000000 | // GENERIC_ALL
		0x40000000 // GENERIC_WRITE
)

type aclSizeInformation struct {
	aceCount     uint32
	aclBytesUsed uint32
	aclBytesFree uint32
}

var (
	advapi32                       = syscall.NewLazyDLL("advapi32.dll")
	getFileSecurityProc            = advapi32.NewProc("GetFileSecurityW")
	getSecurityDescriptorOwnerProc = advapi32.NewProc("GetSecurityDescriptorOwner")
	getSecurityDescriptorDACLProc  = advapi32.NewProc("GetSecurityDescriptorDacl")
	getACLInformationProc          = advapi32.NewProc("GetAclInformation")
	getACEProc                     = advapi32.NewProc("GetAce")
	equalSidProc                   = advapi32.NewProc("EqualSid")
)

// lockDirectoryOwnedByCurrentUser verifies the Windows security descriptor
// owner against the current process token. os.FileInfo does not carry a
// Windows owner, so the security descriptor API is used instead.
func lockDirectoryOwnedByCurrentUser(path string, _ os.FileInfo) (bool, error) {
	return windowsPathOwnedByCurrentUser(path)
}

func currentProcessUserSID() (*syscall.SID, error) {
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return nil, err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return nil, err
	}
	return user.User.Sid, nil
}

func windowsPathOwnedByCurrentUser(path string) (bool, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}

	var needed uint32
	r1, _, firstErr := getFileSecurityProc.Call(
		uintptr(unsafe.Pointer(name)),
		securityInformation,
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
		securityInformation,
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

	userSID, err := currentProcessUserSID()
	if err != nil {
		return false, err
	}

	equal, _, _ := equalSidProc.Call(
		uintptr(unsafe.Pointer(owner)),
		uintptr(unsafe.Pointer(userSID)),
	)
	if equal == 0 {
		return false, nil
	}
	if err := verifyLockDirectoryDACL(descriptor, userSID); err != nil {
		return false, err
	}
	return true, nil
}

func verifyLockDirectoryDACL(descriptor []byte, userSID *syscall.SID) error {
	var present int32
	var dacl uintptr
	var daclDefaulted int32
	r1, _, err := getSecurityDescriptorDACLProc.Call(
		uintptr(unsafe.Pointer(&descriptor[0])),
		uintptr(unsafe.Pointer(&present)),
		uintptr(unsafe.Pointer(&dacl)),
		uintptr(unsafe.Pointer(&daclDefaulted)),
	)
	if r1 == 0 {
		return err
	}
	if present == 0 || dacl == 0 {
		return fmt.Errorf("lock directory has an unrestricted DACL")
	}

	var size aclSizeInformation
	r1, _, err = getACLInformationProc.Call(
		dacl,
		uintptr(unsafe.Pointer(&size)),
		uintptr(unsafe.Sizeof(size)),
		aclInformationSize,
	)
	if r1 == 0 {
		return err
	}

	allowedSIDs, err := lockDirectoryAllowedSIDs(userSID)
	if err != nil {
		return err
	}
	for index := uint32(0); index < size.aceCount; index++ {
		var ace uintptr
		r1, _, err = getACEProc.Call(
			dacl,
			uintptr(index),
			uintptr(unsafe.Pointer(&ace)),
		)
		if r1 == 0 {
			return err
		}
		if ace == 0 {
			return fmt.Errorf("lock directory DACL contains a nil ACE")
		}

		aceType := *(*byte)(unsafe.Pointer(ace))
		switch aceType {
		case accessDeniedACEType, accessDeniedCallbackACEType, accessDeniedObjectACEType, accessDeniedCallbackObjectACEType:
			continue
		case accessAllowedACEType:
		default:
			return fmt.Errorf("lock directory DACL contains unsupported ACE type %d", aceType)
		}
		mask := *(*uint32)(unsafe.Pointer(ace + 4))
		if mask&lockDirectoryWriteMask == 0 {
			continue
		}
		aceSID := (*syscall.SID)(unsafe.Pointer(ace + 8))
		if !sidMatchesAny(aceSID, allowedSIDs) {
			return fmt.Errorf("lock directory DACL grants write access to an unauthorized principal")
		}
	}
	return nil
}

func lockDirectoryAllowedSIDs(userSID *syscall.SID) ([]*syscall.SID, error) {
	allowed := []*syscall.SID{userSID}
	for _, value := range []string{"S-1-5-18", "S-1-5-32-544", "S-1-3-0"} {
		sid, err := syscall.StringToSid(value)
		if err != nil {
			return nil, err
		}
		allowed = append(allowed, sid)
	}
	return allowed, nil
}

func sidMatchesAny(candidate *syscall.SID, allowed []*syscall.SID) bool {
	for _, sid := range allowed {
		equal, _, _ := equalSidProc.Call(
			uintptr(unsafe.Pointer(candidate)),
			uintptr(unsafe.Pointer(sid)),
		)
		if equal != 0 {
			return true
		}
	}
	return false
}
