/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"errors"
	"os"
)

// FileIdentity is an opaque identity captured from an opened filesystem
// entry. It intentionally exposes no operating-system-specific metadata while
// still allowing callers to detect replacement, symlink retargeting, and
// hard-link aliases through os.SameFile.
type FileIdentity struct {
	info os.FileInfo
}

// NewFileIdentity captures the identity represented by info. A nil info
// produces an invalid identity.
func NewFileIdentity(info os.FileInfo) FileIdentity {
	return FileIdentity{info: info}
}

// Valid reports whether the identity contains a captured filesystem entry.
func (id FileIdentity) Valid() bool {
	return id.info != nil
}

// Same reports whether two captured identities refer to the same filesystem
// entry. It recognizes hard-link aliases on platforms supported by os.SameFile.
func (id FileIdentity) Same(other FileIdentity) bool {
	if !id.Valid() || !other.Valid() {
		return false
	}
	return os.SameFile(id.info, other.info)
}

// MatchesPath reports whether path still identifies the captured filesystem
// entry. The path is statted through the operating system, so symlink target
// changes and file replacement are detected.
func (id FileIdentity) MatchesPath(path string) (bool, error) {
	if !id.Valid() {
		return false, errors.New("file identity is invalid")
	}

	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return os.SameFile(id.info, info), nil
}
