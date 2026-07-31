/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package dotenv

import (
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/envaar/vaar/internal/envfile"
)

// Document is one selected dotenv source together with the parsed model and
// file metadata required by later analysis and mutation layers.
//
// File remains embedded temporarily so source consumers can use the existing
// parser facts without duplicating the envfile model. SourcePath identifies
// the on-disk input, while File.Path retains the caller-provided display path.
type Document struct {
	envfile.File
	SourcePath string
	Mode       os.FileMode
}

// Load validates, reads and parses one selected dotenv file. displayPath is
// preserved as the parsed document path for diagnostics and reports.
func Load(path, displayPath string) (Document, error) {
	// O_NONBLOCK prevents opening a Unix FIFO from blocking before its type can
	// be checked. Windows ignores this flag because its file handles are already
	// non-blocking.
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return Document{}, fmt.Errorf("open dotenv source %q: %w", path, err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return Document{}, fmt.Errorf("stat dotenv source %q: %w", path, err)
	}

	if info.IsDir() {
		_ = file.Close()
		return Document{}, fmt.Errorf("dotenv source %q is a directory", path)
	}
	if !info.Mode().IsRegular() {
		_ = file.Close()
		return Document{}, fmt.Errorf("dotenv source %q is not a regular file", path)
	}

	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return Document{}, fmt.Errorf("read dotenv source %q: %w", path, errors.Join(readErr, closeErr))
	}
	if closeErr != nil {
		return Document{}, fmt.Errorf("close dotenv source %q: %w", path, closeErr)
	}

	parsed, err := envfile.Parse(displayPath, data)
	if err != nil {
		return Document{}, fmt.Errorf("parse dotenv source %q: %w", displayPath, err)
	}

	return Document{
		File:       parsed,
		SourcePath: path,
		Mode:       info.Mode().Perm(),
	}, nil
}

// LoadMany loads selected dotenv files in the order supplied by paths and
// preserves the corresponding display path at each position. It returns no
// partial document set when any input fails.
func LoadMany(paths, displayPaths []string) ([]Document, error) {
	if len(paths) != len(displayPaths) {
		return nil, fmt.Errorf(
			"dotenv load requires one display path per source path: got %d paths and %d display paths",
			len(paths), len(displayPaths),
		)
	}

	documents := make([]Document, 0, len(paths))
	for i, path := range paths {
		document, err := Load(path, displayPaths[i])
		if err != nil {
			return nil, fmt.Errorf("load dotenv document %q: %w", displayPaths[i], err)
		}
		documents = append(documents, document)
	}

	return documents, nil
}
