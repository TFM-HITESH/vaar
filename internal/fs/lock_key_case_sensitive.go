//go:build !darwin && !windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

func normalizeLockPath(path string) string {
	return path
}
