/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package scope

import (
	"fmt"
	"os"

	"github.com/envaar/vaar/internal/fs"
)

// ValidateOutputPath reports an error if outputPath resolves to the same file
// as any input path in the resolved selection. The comparison uses canonical
// absolute paths and file identity so relative, cleaned, symlink-equivalent
// and case-insensitive aliases are detected.
func ValidateOutputPath(selection Selection, outputPath string) error {
	out, err := fs.CanonicalPath(outputPath)
	if err != nil {
		return fmt.Errorf("resolve output path %q: %w", outputPath, err)
	}

	for _, input := range selection.Paths {
		in, err := fs.CanonicalPath(input)
		if err != nil {
			return fmt.Errorf("resolve input path %q: %w", input, err)
		}
		if out == in || sameExistingFile(outputPath, input) {
			return fmt.Errorf("cannot write lint output to %q: the path is also a lint input file", outputPath)
		}
	}

	return nil
}

// sameExistingFile reports whether two existing paths identify the same file.
// It intentionally ignores stat errors: CanonicalPath already owns path
// resolution errors, while a not-yet-created output is a valid destination.
func sameExistingFile(firstPath, secondPath string) bool {
	first, err := os.Stat(firstPath)
	if err != nil {
		return false
	}
	second, err := os.Stat(secondPath)
	if err != nil {
		return false
	}
	return os.SameFile(first, second)
}
