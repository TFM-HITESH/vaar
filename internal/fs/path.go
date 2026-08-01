/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNotRegularFile indicates that a path does not identify a regular file.
var ErrNotRegularFile = errors.New("path is not a regular file")

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

// ValidateRegularFile returns metadata for a readable regular file.
func ValidateRegularFile(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s", ErrNotRegularFile, path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	if err := file.Close(); err != nil {
		return nil, err
	}
	return info, nil
}

// ReadFile validates path as a regular file and returns its original bytes.
func ReadFile(path string) ([]byte, error) {
	if _, err := ValidateRegularFile(path); err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}
