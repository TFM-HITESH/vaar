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

// AtomicFile owns a same-directory temporary file and destination writer lock
// until it is finalized or cleaned up. The destination is replaced only after
// all writes succeed.
type AtomicFile struct {
	destination string
	temporary   string
	file        *os.File
	lock        *pathLock
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

	lock, err := acquirePathLock(destination)
	if err != nil {
		return nil, err
	}
	if err := ValidateFileDestination(destination); err != nil {
		return nil, errors.Join(err, lock.release())
	}

	temporary, err := os.CreateTemp(TempDirForPath(destination), "vaar-atomic-*")
	if err != nil {
		return nil, errors.Join(err, lock.release())
	}

	return &AtomicFile{
		destination: destination,
		temporary:   temporary.Name(),
		file:        temporary,
		lock:        lock,
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
		return written, errors.Join(err, f.Cleanup())
	}
	if written != len(data) {
		return written, errors.Join(io.ErrShortWrite, f.Cleanup())
	}
	return written, nil
}

// Chmod applies permission bits to the temporary replacement file. The
// destination is unchanged until Finalize succeeds, so callers can preserve
// an existing destination mode as part of an atomic replacement.
func (f *AtomicFile) Chmod(mode os.FileMode) error {
	if f == nil || f.file == nil {
		return ErrAtomicFileClosed
	}

	if err := f.file.Chmod(mode.Perm()); err != nil {
		return errors.Join(err, f.Cleanup())
	}
	return nil
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
			return errors.Join(err, f.Cleanup())
		}
	}

	if err := replaceFile(f.temporary, f.destination); err != nil {
		return errors.Join(err, f.Cleanup())
	}

	f.temporary = ""
	f.finalized = true
	return f.Cleanup()
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

	var removeErr error
	if f.temporary != "" {
		removeErr = os.Remove(f.temporary)
		if removeErr == nil || os.IsNotExist(removeErr) {
			f.temporary = ""
		}
	}

	var unlockErr error
	if f.lock != nil {
		unlockErr = f.lock.release()
		f.lock = nil
	}
	return errors.Join(closeErr, removeErr, unlockErr)
}

func replaceFile(temporary, destination string) error {
	// Revalidate immediately before replacement so a directory or a symlink to
	// a directory created after NewAtomicFile cannot be replaced as a file.
	if err := ValidateFileDestination(destination); err != nil {
		return err
	}

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

// replaceFileWindows preserves the original destination while replacing a
// regular file on Windows, where a direct rename over an existing file may
// fail because the destination is still open.
func replaceFileWindows(temporary, destination string) error {
	if err := ValidateFileDestination(destination); err != nil {
		return err
	}

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

// hasTrailingSeparator reports whether path denotes a directory-like target
// through a trailing platform-independent or native path separator.
func hasTrailingSeparator(path string) bool {
	if path == "" {
		return false
	}
	last := path[len(path)-1]
	return last == '/' || last == os.PathSeparator
}

// directoryDestinationError preserves the directory identity for callers that
// need to translate it into a user-facing error.
func directoryDestinationError(path string) error {
	return fmt.Errorf("%w: %s", ErrIsDirectory, path)
}
