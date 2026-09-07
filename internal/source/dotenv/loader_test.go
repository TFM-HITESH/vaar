/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package dotenv_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/envaar/vaar/internal/source/dotenv"
)

func TestLoadParsesAndPreservesSourceMetadata(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "selected.env")
	data := []byte("FOO=bar\nEMPTY=\n")
	mustWrite(t, path, data)

	const displayPath = "service/config.env"
	document, err := dotenv.Load(path, displayPath)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if got, want := document.SourcePath, path; got != want {
		t.Fatalf("unexpected source path: got %q want %q", got, want)
	}
	if got, want := document.Path, displayPath; got != want {
		t.Fatalf("unexpected display path: got %q want %q", got, want)
	}
	if !bytes.Equal(document.Original, data) {
		t.Fatalf("original bytes changed: got %q want %q", document.Original, data)
	}
	if got, want := len(document.Lines), 2; got != want {
		t.Fatalf("unexpected parsed line count: got %d want %d", got, want)
	}
	if got, want := document.Lines[0].Key, "FOO"; got != want {
		t.Fatalf("unexpected first key: got %q want %q", got, want)
	}
	if got, want := document.Lines[0].Value, "bar"; got != want {
		t.Fatalf("unexpected first value: got %q want %q", got, want)
	}
}

func TestLoadPreservesEmptyDocument(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	mustWrite(t, path, nil)

	document, err := dotenv.Load(path, ".env")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if len(document.Lines) != 0 {
		t.Fatalf("expected no parsed lines, got %d", len(document.Lines))
	}
	if len(document.Original) != 0 {
		t.Fatalf("expected empty original bytes, got %q", document.Original)
	}
}

func TestLoadPreservesFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	mustWrite(t, path, []byte("KEY=value\n"))
	if err := os.Chmod(path, 0o640); err != nil {
		t.Skipf("file mode is not supported: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	document, err := dotenv.Load(path, ".env")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if got, want := document.Mode, info.Mode().Perm(); got != want {
		t.Fatalf("unexpected mode: got %04o want %04o", got, want)
	}
}

func TestLoadCapturesResolvedTargetAndIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file symlink behavior is not portable on Windows")
	}

	root := t.TempDir()
	target := filepath.Join(root, "target.env")
	alias := filepath.Join(root, "alias.env")
	mustWrite(t, target, []byte("KEY=value\n"))
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	document, err := dotenv.Load(alias, ".env")
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if got, want := document.ResolvedPath, target; got != want {
		t.Fatalf("resolved path = %q, want %q", got, want)
	}
	if !document.Identity.Valid() {
		t.Fatal("loaded document has no file identity")
	}
	matched, err := document.Identity.MatchesPath(target)
	if err != nil {
		t.Fatalf("match target identity failed: %v", err)
	}
	if !matched {
		t.Fatal("loaded identity does not match resolved target")
	}
}

func TestLoadManyPreservesInputOrderAndDisplayPaths(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "first.env")
	secondPath := filepath.Join(root, "second.env")
	mustWrite(t, firstPath, []byte("FIRST=one\n"))
	mustWrite(t, secondPath, []byte("SECOND=two\n"))

	paths := []string{secondPath, firstPath}
	displayPaths := []string{"service/second.env", "service/first.env"}
	documents, err := dotenv.LoadMany(paths, displayPaths)
	if err != nil {
		t.Fatalf("batch load failed: %v", err)
	}

	if got, want := len(documents), len(paths); got != want {
		t.Fatalf("unexpected document count: got %d want %d", got, want)
	}
	for i := range paths {
		if got, want := documents[i].SourcePath, paths[i]; got != want {
			t.Fatalf("unexpected source path at %d: got %q want %q", i, got, want)
		}
		if got, want := documents[i].Path, displayPaths[i]; got != want {
			t.Fatalf("unexpected display path at %d: got %q want %q", i, got, want)
		}
	}
}

func TestLoadManyRejectsMismatchedPathListsBeforeReading(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.env")

	_, err := dotenv.LoadMany([]string{missingPath}, nil)
	if err == nil {
		t.Fatal("expected mismatched path-list error")
	}
	if !strings.Contains(err.Error(), "display") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadManyReturnsNoPartialDocumentsAfterFailure(t *testing.T) {
	root := t.TempDir()
	validPath := filepath.Join(root, "valid.env")
	missingPath := filepath.Join(root, "missing.env")
	mustWrite(t, validPath, []byte("KEY=value\n"))

	documents, err := dotenv.LoadMany(
		[]string{validPath, missingPath},
		[]string{"valid.env", "missing.env"},
	)
	if err == nil {
		t.Fatal("expected batch load error")
	}
	if documents != nil {
		t.Fatalf("expected no partial documents, got %#v", documents)
	}
}

func TestLoadRejectsMissingPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.env")

	_, err := dotenv.Load(path, ".env")
	if err == nil {
		t.Fatal("expected missing-path error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected not-exist error, got %v", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Fatalf("error does not identify path: %v", err)
	}
}

func TestLoadRejectsDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dotenv-source-directory")
	if err := os.Mkdir(path, 0o755); err != nil {
		t.Fatalf("create directory failed: %v", err)
	}

	_, err := dotenv.Load(path, ".env")
	if err == nil {
		t.Fatal("expected directory error")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Fatalf("error does not identify directory input: %v", err)
	}
	if !strings.Contains(err.Error(), filepath.Base(path)) {
		t.Fatalf("error does not identify path: %v", err)
	}
}

func TestLoadRejectsNonRegularFileWhereSupported(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.DevNull is not a regular filesystem entry on all supported Windows environments")
	}

	info, err := os.Stat(os.DevNull)
	if err != nil || info.Mode().IsRegular() {
		t.Skip("the platform does not expose a non-regular os.DevNull entry")
	}

	_, err = dotenv.Load(os.DevNull, os.DevNull)
	if err == nil {
		t.Fatal("expected non-regular-file error")
	}
	if !strings.Contains(err.Error(), "regular") {
		t.Fatalf("error does not identify non-regular input: %v", err)
	}
}

func TestLoadRejectsUnreadableFileWhereSupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unreadable.env")
	mustWrite(t, path, []byte("KEY=value\n"))
	if err := os.Chmod(path, 0); err != nil {
		t.Skipf("file permissions are not supported: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })

	_, err := dotenv.Load(path, ".env")
	if err == nil {
		t.Skip("the test process can read mode-000 files")
	}
	if !errors.Is(err, os.ErrPermission) {
		t.Fatalf("expected permission error, got %v", err)
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write failed: %v", err)
	}
}
