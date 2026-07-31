/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package scope

import (
	"fmt"

	"github.com/envaar/vaar/internal/fs"
)

// ValidateOutputPath reports an error if outputPath resolves to the same file
// as any input path in the resolved selection. The comparison uses canonical
// absolute paths so relative, cleaned and symlink-equivalent forms are
// detected.
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
		if out == in {
			return fmt.Errorf("cannot write lint output to %q: the path is also a lint input file", outputPath)
		}
	}

	return nil
}
