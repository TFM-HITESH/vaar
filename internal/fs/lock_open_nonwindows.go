//go:build !windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import "os"

func openTargetFile(path string, flags int) (*os.File, error) {
	return os.OpenFile(path, flags, 0)
}
