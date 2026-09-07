//go:build !aix && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !solaris && !windows

/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"errors"
	"os"
	"testing"
)

func TestLockFileReportsUnsupportedPlatform(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "lock-")
	if err != nil {
		t.Fatalf("create lock fixture failed: %v", err)
	}
	defer file.Close()

	if err := lockFile(file); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("lockFile error = %v, want errors.ErrUnsupported", err)
	}
	if err := unlockFile(file); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("unlockFile error = %v, want errors.ErrUnsupported", err)
	}
}
