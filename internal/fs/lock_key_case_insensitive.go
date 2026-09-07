//go:build darwin || windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import "strings"

// normalizeLockPath applies the path-name equivalence used by the default
// case-insensitive filesystems on macOS and Windows. Lower-casing a missing
// destination is necessary because there is no filesystem entry for
// CanonicalPath to resolve yet.
func normalizeLockPath(path string) string {
	return strings.ToLower(path)
}
