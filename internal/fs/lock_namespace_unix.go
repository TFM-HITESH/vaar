//go:build aix || darwin || dragonfly || freebsd || illumos || linux || netbsd || openbsd || solaris

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"fmt"
	"os"
)

func fallbackLockNamespace() (string, error) {
	return fmt.Sprintf("uid:%d", os.Geteuid()), nil
}
