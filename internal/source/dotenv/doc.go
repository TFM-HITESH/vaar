/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

// Package dotenv loads selected dotenv files into source-owned documents.
//
// It owns dotenv parsing, source-specific syntax facts, and pure byte
// transformations used by dotenv fixes. Filesystem mechanics remain in
// internal/fs, while analysis adapters must discard raw values and bytes.
package dotenv
