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
	"syscall"
)

// ErrNotRegularFile indicates that a path does not identify a regular file.
var ErrNotRegularFile = errors.New("path is not a regular file")

// ErrIsDirectory identifies the directory case of ErrNotRegularFile so
// callers can preserve directory-specific error wording.
var ErrIsDirectory = errors.New("path is a directory")

// ResolvePath returns path as a cleaned absolute path. Relative paths are
// anchored to root, while an empty root uses the current working directory.
func ResolvePath(root, path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	if root == "" {
		root = "."
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	return filepath.Join(absoluteRoot, path), nil
}

// CanonicalPath returns a stable absolute form of path. It resolves symlinks
// and cleans . and .. segments. If path does not exist yet, the parent
// directory is canonicalized and the base name is rejoined so the result can
// still be compared with existing files.
func CanonicalPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	dir := filepath.Dir(abs)
	resolvedDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return "", err
	}
	return filepath.Join(resolvedDir, filepath.Base(abs)), nil
}

// ValidateRegularFile returns metadata for a readable regular file. The
// metadata returned is from the descriptor that was opened for validation.
func ValidateRegularFile(path string) (os.FileInfo, error) {
	file, info, err := openRegularFile(path)
	if err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return info, nil
}

// ReadFile validates path as a regular file and returns its original bytes
// from the same descriptor used for validation.
func ReadFile(path string) (data []byte, err error) {
	file, _, err := openRegularFile(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = errors.Join(err, file.Close())
	}()
	return io.ReadAll(file)
}

func openRegularFile(path string) (*os.File, os.FileInfo, error) {
	pathInfo, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	if err := validateRegularFileInfo(pathInfo, path); err != nil {
		return nil, nil, err
	}

	// O_NONBLOCK prevents a TOCTOU replacement with a FIFO from blocking the
	// process between the pathname check and descriptor validation. Windows
	// ignores this flag because its file handles are already non-blocking.
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, nil, err
	}

	info, err := file.Stat()
	if err != nil {
		return nil, nil, errors.Join(err, file.Close())
	}
	if err := validateRegularFileInfo(info, path); err != nil {
		return nil, nil, errors.Join(err, file.Close())
	}
	return file, info, nil
}

func validateRegularFileInfo(info os.FileInfo, path string) error {
	if info.IsDir() {
		return fmt.Errorf("%w: %w", ErrNotRegularFile, ErrIsDirectory)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: %s", ErrNotRegularFile, path)
	}
	return nil
}
