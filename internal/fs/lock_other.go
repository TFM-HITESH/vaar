//go:build !aix && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !solaris && !windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"errors"
	"os"
)

func lockFile(*os.File) error { return errors.ErrUnsupported }

func unlockFile(*os.File) error { return errors.ErrUnsupported }
