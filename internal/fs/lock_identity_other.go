//go:build !aix && (!solaris || illumos)

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"errors"
	"os"
)

func openIdentityTargetLock(string) (*os.File, error) {
	return nil, errors.ErrUnsupported
}
