/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package fs_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/envaar/vaar/internal/fs"
)

func TestResolvePathAnchorsRelativePathsAndCleansAbsolutePaths(t *testing.T) {
	root := t.TempDir()

	got, err := fs.ResolvePath(root, filepath.Join("nested", "..", "config.env"))
	if err != nil {
		t.Fatalf("resolve relative path failed: %v", err)
	}
	if want := filepath.Join(root, "config.env"); got != want {
		t.Fatalf("unexpected relative path: got %q want %q", got, want)
	}

	absolute := filepath.Join(root, "nested", "..", "absolute.env")
	got, err = fs.ResolvePath(root, absolute)
	if err != nil {
		t.Fatalf("resolve absolute path failed: %v", err)
	}
	if want := filepath.Join(root, "absolute.env"); got != want {
		t.Fatalf("unexpected absolute path: got %q want %q", got, want)
	}
}

func TestResolvePathUsesWorkingDirectoryForEmptyRoot(t *testing.T) {
	root := t.TempDir()
	withWorkingDir(t, root)

	got, err := fs.ResolvePath("", "config.env")
	if err != nil {
		t.Fatalf("resolve empty-root path failed: %v", err)
	}
	want, err := filepath.Abs("config.env")
	if err != nil {
		t.Fatalf("resolve expected empty-root path failed: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected empty-root path: got %q want %q", got, want)
	}
}

func TestCanonicalPathResolvesExistingAndMissingPaths(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "config.env")
	if err := os.WriteFile(input, []byte("KEY=value\n"), 0o644); err != nil {
		t.Fatalf("write input failed: %v", err)
	}

	got, err := fs.CanonicalPath(filepath.Join(root, "nested", "..", "config.env"))
	if err != nil {
		t.Fatalf("canonicalize existing path failed: %v", err)
	}
	want, err := filepath.EvalSymlinks(input)
	if err != nil {
		t.Fatalf("resolve expected existing path symlinks failed: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected existing canonical path: got %q want %q", got, want)
	}

	reports := filepath.Join(root, "reports")
	if err := os.Mkdir(reports, 0o755); err != nil {
		t.Fatalf("create reports directory failed: %v", err)
	}
	missing := filepath.Join(reports, "lint.json")
	got, err = fs.CanonicalPath(missing)
	if err != nil {
		t.Fatalf("canonicalize missing path failed: %v", err)
	}
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(missing))
	if err != nil {
		t.Fatalf("resolve expected missing parent symlinks failed: %v", err)
	}
	want = filepath.Join(resolvedParent, filepath.Base(missing))
	if got != want {
		t.Fatalf("unexpected missing canonical path: got %q want %q", got, want)
	}
}

func TestCanonicalPathResolvesSymlinkEquivalentPaths(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "config.env")
	if err := os.WriteFile(input, []byte("KEY=value\n"), 0o644); err != nil {
		t.Fatalf("write input failed: %v", err)
	}

	alias := filepath.Join(root, "alias.env")
	if err := os.Symlink(input, alias); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("symlink creation is unavailable: %v", err)
		}
		t.Fatalf("create symlink failed: %v", err)
	}

	got, err := fs.CanonicalPath(alias)
	if err != nil {
		t.Fatalf("canonicalize symlink failed: %v", err)
	}
	want, err := filepath.EvalSymlinks(input)
	if err != nil {
		t.Fatalf("resolve expected symlink target failed: %v", err)
	}
	if got != want {
		t.Fatalf("unexpected symlink canonical path: got %q want %q", got, want)
	}
}

func TestCanonicalPathReportsMissingParent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "lint.json")

	_, err := fs.CanonicalPath(path)
	if err == nil {
		t.Fatal("expected canonicalization to fail for a missing parent")
	}
}

func TestValidateRegularFileReturnsMetadataForReadableFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.env")
	if err := os.WriteFile(path, []byte("KEY=value\n"), 0o640); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	info, err := fs.ValidateRegularFile(path)
	if err != nil {
		t.Fatalf("validate regular file failed: %v", err)
	}
	if !info.Mode().IsRegular() {
		t.Fatalf("validated mode is not regular: %v", info.Mode())
	}
}

func TestValidateRegularFileRejectsDirectory(t *testing.T) {
	path := t.TempDir()

	_, err := fs.ValidateRegularFile(path)
	if !errors.Is(err, fs.ErrNotRegularFile) {
		t.Fatalf("unexpected directory error: %v", err)
	}
}

func TestValidateRegularFileRejectsNonRegularFileWhereSupported(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("device files are not portable on windows")
	}

	if _, err := os.Stat("/dev/null"); err != nil {
		t.Skipf("device file is unavailable: %v", err)
	}

	_, err := fs.ValidateRegularFile("/dev/null")
	if !errors.Is(err, fs.ErrNotRegularFile) {
		t.Fatalf("unexpected non-regular-file error: %v", err)
	}
}

func TestValidateRegularFileReportsUnreadableFileWhereSupported(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission fixtures are not portable on windows")
	}

	path := filepath.Join(t.TempDir(), "locked.env")
	if err := os.WriteFile(path, []byte("KEY=value\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, 0o644); err != nil {
			t.Errorf("restore permissions failed: %v", err)
		}
	})
	if err := os.Chmod(path, 0); err != nil {
		t.Fatalf("lock file failed: %v", err)
	}

	_, err := fs.ValidateRegularFile(path)
	if err == nil {
		t.Skip("file permissions are not enforceable for this test user")
	}
	if !os.IsPermission(err) {
		t.Fatalf("unexpected unreadable-file error: %v", err)
	}
}

func TestReadFilePreservesBytes(t *testing.T) {
	cases := []struct {
		name string
		data []byte
	}{
		{name: "normal", data: []byte("KEY=value\r\nSECOND=\"quoted\"\n")},
		{name: "empty", data: []byte{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.env")
			if err := os.WriteFile(path, tc.data, 0o644); err != nil {
				t.Fatalf("write file failed: %v", err)
			}

			got, err := fs.ReadFile(path)
			if err != nil {
				t.Fatalf("read file failed: %v", err)
			}
			if string(got) != string(tc.data) {
				t.Fatalf("bytes changed: got %q want %q", got, tc.data)
			}
		})
	}
}

func TestReadFileRejectsDirectory(t *testing.T) {
	_, err := fs.ReadFile(t.TempDir())
	if !errors.Is(err, fs.ErrNotRegularFile) {
		t.Fatalf("unexpected directory read error: %v", err)
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Errorf("restore working dir failed: %v", err)
		}
	})
}
