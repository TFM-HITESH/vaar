/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package envfile

import "github.com/envaar/vaar/internal/fs"

// Write writes data back to path and preserves the file's existing permissions
// when the file already exists.
func Write(path string, data []byte) error {
	return fs.WriteFile(path, data)
}
