//go:build aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"fmt"
	"os"
	"syscall"
)

func lockDirectoryOwnedByCurrentUser(_ string, info os.FileInfo) (bool, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("lock directory has unsupported ownership metadata")
	}
	return stat.Uid == uint32(os.Geteuid()), nil
}
