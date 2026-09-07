//go:build !aix && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !solaris && !windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import "os"

func lockDirectoryOwnedByCurrentUser(string, os.FileInfo) (bool, error) {
	// These platforms do not expose a portable ownership representation in
	// the standard library. The directory remains protected by its private
	// namespace and the platform's normal filesystem permission checks.
	return true, nil
}
