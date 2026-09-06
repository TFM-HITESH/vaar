/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package cli

import (
	"errors"
	"fmt"

	"github.com/envaar/vaar/internal/fs"
)

// validateOutputDestination rejects --output values that name a directory and
// preserves the CLI-specific directory error before lint execution begins.
// The filesystem transaction repeats this safety check at finalization for
// destinations that change after this preflight.
func validateOutputDestination(output string) error {
	if err := fs.ValidateFileDestination(output); err != nil {
		if errors.Is(err, fs.ErrIsDirectory) {
			return outputDirectoryError(output)
		}
		return NewToolError(fmt.Sprintf("checking output destination %q failed", output), err)
	}
	return nil
}

// writeJSONOutput keeps the lint-specific error wording at the CLI boundary
// while delegating temporary-file creation, replacement and cleanup to fs.
func writeJSONOutput(path string, data []byte) error {
	file, err := fs.NewAtomicFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrIsDirectory) {
			return outputDirectoryError(path)
		}
		return NewToolError("creating JSON output file failed", err)
	}
	defer func() { _ = file.Cleanup() }()

	if _, err := file.Write(data); err != nil {
		return NewToolError(fmt.Sprintf("writing JSON output to %s failed", path), err)
	}
	if err := file.Finalize(); err != nil {
		if errors.Is(err, fs.ErrIsDirectory) {
			return outputDirectoryError(path)
		}
		return outputWriteError(path, err)
	}
	return nil
}

// outputTempDir keeps temporary JSON output alongside the final destination so
// the eventual rename stays on the same volume.
func outputTempDir(path string) string {
	return fs.TempDirForPath(path)
}

func outputDirectoryError(path string) error {
	return NewToolError(fmt.Sprintf("cannot write lint output to %q: the path is a directory", path), nil)
}

func outputWriteError(path string, err error) error {
	return NewToolError(fmt.Sprintf("writing JSON output to %s failed", path), err)
}
