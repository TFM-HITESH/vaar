/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

// Package dotenv loads selected dotenv files into source documents.
//
// The source boundary owns filesystem loading and delegates the temporary
// parser/model implementation to internal/envfile while downstream layers
// migrate to source-owned representations.
package dotenv
