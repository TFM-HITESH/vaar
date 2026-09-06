/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// ErrAtomicFileClosed indicates that an AtomicFile has already been
// finalized or cleaned up.
var ErrAtomicFileClosed = errors.New("atomic file is closed")

// AtomicFile owns a same-directory temporary file until it is finalized or
// cleaned up. The destination is replaced only after all writes succeed.
type AtomicFile struct {
	destination string
	temporary   string
	file        *os.File
	finalized   bool
}

// TempDirForPath returns the directory in which a temporary file for path
// must be created so a subsequent rename remains on the same volume.
func TempDirForPath(path string) string {
	return filepath.Dir(path)
}

// ValidateFileDestination rejects paths that name directories and allows paths
// that do not exist yet. Errors other than a missing path are returned to the
// caller unchanged.
func ValidateFileDestination(path string) error {
	if hasTrailingSeparator(path) {
		return directoryDestinationError(path)
	}

	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.IsDir() {
			return directoryDestinationError(path)
		}
		return nil
	case os.IsNotExist(err):
		return nil
	default:
		return err
	}
}

// NewAtomicFile creates a transaction for replacing destination. The
// temporary file is created in destination's directory and uses a generic
// name so the helper remains independent of its caller's output format.
func NewAtomicFile(destination string) (*AtomicFile, error) {
	if err := ValidateFileDestination(destination); err != nil {
		return nil, err
	}

	temporary, err := os.CreateTemp(TempDirForPath(destination), "vaar-atomic-*")
	if err != nil {
		return nil, err
	}

	return &AtomicFile{
		destination: destination,
		temporary:   temporary.Name(),
		file:        temporary,
	}, nil
}

// Write writes data to the transaction's temporary file. The destination is
// not changed until Finalize succeeds.
func (f *AtomicFile) Write(data []byte) (int, error) {
	if f == nil || f.file == nil {
		return 0, ErrAtomicFileClosed
	}

	written, err := f.file.Write(data)
	if err != nil {
		_ = f.Cleanup()
		return written, err
	}
	if written != len(data) {
		_ = f.Cleanup()
		return written, io.ErrShortWrite
	}
	return written, nil
}

// Finalize closes the temporary file and atomically replaces the destination.
// It cleans up the temporary file if closing or replacement fails.
func (f *AtomicFile) Finalize() error {
	if f == nil {
		return ErrAtomicFileClosed
	}
	if f.finalized {
		return nil
	}
	if f.temporary == "" {
		return ErrAtomicFileClosed
	}

	if f.file != nil {
		err := f.file.Close()
		f.file = nil
		if err != nil {
			_ = f.Cleanup()
			return err
		}
	}

	if err := replaceFile(f.temporary, f.destination); err != nil {
		_ = f.Cleanup()
		return err
	}

	f.temporary = ""
	f.finalized = true
	return nil
}

// Cleanup closes any open temporary file and removes an uncommitted temporary
// path. It is safe to call repeatedly and after successful finalization.
func (f *AtomicFile) Cleanup() error {
	if f == nil {
		return nil
	}

	var closeErr error
	if f.file != nil {
		closeErr = f.file.Close()
		f.file = nil
	}

	if f.temporary == "" {
		return closeErr
	}

	removeErr := os.Remove(f.temporary)
	if removeErr == nil || os.IsNotExist(removeErr) {
		f.temporary = ""
	}
	return errors.Join(closeErr, removeErr)
}

func replaceFile(temporary, destination string) error {
	renameErr := os.Rename(temporary, destination)
	if renameErr == nil {
		return nil
	}

	info, statErr := os.Stat(destination)
	if statErr == nil {
		if info.IsDir() {
			return directoryDestinationError(destination)
		}
		if runtime.GOOS == "windows" {
			return replaceFileWindows(temporary, destination)
		}
	}

	return renameErr
}

func replaceFileWindows(temporary, destination string) error {
	backup, err := os.CreateTemp(TempDirForPath(destination), filepath.Base(destination)+".vaar-backup-*")
	if err != nil {
		return err
	}
	backupName := backup.Name()
	if err := backup.Close(); err != nil {
		_ = os.Remove(backupName)
		return err
	}

	cleanBackup := true
	defer func() {
		if cleanBackup {
			_ = os.Remove(backupName)
		}
	}()

	if err := os.Rename(destination, backupName); err != nil {
		if chmodErr := os.Chmod(destination, 0o600); chmodErr == nil {
			if retryErr := os.Rename(destination, backupName); retryErr == nil {
				err = nil
			} else {
				err = retryErr
			}
		}
		if err != nil {
			return err
		}
	}

	if err := os.Rename(temporary, destination); err != nil {
		if restoreErr := os.Rename(backupName, destination); restoreErr != nil {
			cleanBackup = false
			return fmt.Errorf("%w; restore original failed: %v", err, restoreErr)
		}
		return err
	}

	return nil
}

func hasTrailingSeparator(path string) bool {
	if path == "" {
		return false
	}
	last := path[len(path)-1]
	return last == '/' || last == os.PathSeparator
}

func directoryDestinationError(path string) error {
	return fmt.Errorf("%w: %s", ErrIsDirectory, path)
}
