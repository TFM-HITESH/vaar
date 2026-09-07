//go:build !aix && !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !solaris && !windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import "os"

func lockFile(*os.File) error { return nil }

func unlockFile(*os.File) error { return nil }
