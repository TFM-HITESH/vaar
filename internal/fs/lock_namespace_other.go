//go:build !aix && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !solaris && !windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import "runtime"

func fallbackLockNamespace() (string, error) {
	return "platform:" + runtime.GOOS, nil
}
