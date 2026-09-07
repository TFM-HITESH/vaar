/*
Copyright © 2026 envaar
SPDX-License-Identifier: Apache-2.0
*/

package mutations_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/envaar/vaar/internal/lint"
	"github.com/envaar/vaar/internal/lint/rules"
	"github.com/envaar/vaar/internal/mutations"
	sourcedotenv "github.com/envaar/vaar/internal/source/dotenv"
)

type plainRule struct{}

func (plainRule) ID() string                               { return "plain" }
func (plainRule) Description() string                      { return "no fix half" }
func (plainRule) Run(lint.Context) ([]lint.Finding, error) { return nil, nil }

type markerStripRule struct{}

func (markerStripRule) ID() string { return "zz-custom" }
func (markerStripRule) Description() string {
	return "resolves the marker by line-ending state"
}
func (markerStripRule) Run(lint.Context) ([]lint.Finding, error) { return nil, nil }
func (markerStripRule) Fix(data []byte) []byte {
	if bytes.Contains(data, []byte("\r")) {
		return bytes.ReplaceAll(data, []byte("X"), []byte("crlf"))
	}
	return bytes.ReplaceAll(data, []byte("X"), []byte("lf"))
}

type sequencedFixRule struct {
	calls int
}

func (r *sequencedFixRule) ID() string          { return "sequenced-fix" }
func (r *sequencedFixRule) Description() string { return "test-only sequenced replacement" }
func (r *sequencedFixRule) Run(lint.Context) ([]lint.Finding, error) {
	return nil, nil
}
func (r *sequencedFixRule) Fix([]byte) []byte {
	r.calls++
	if r.calls == 1 {
		return []byte("FIRST=changed\n")
	}
	return []byte("SECOND=changed\n")
}

func TestBuildPlanEmpty(t *testing.T) {
	plan, err := mutations.BuildPlan(nil, nil)
	if err != nil {
		t.Fatalf("build empty plan failed: %v", err)
	}
	if changes := plan.Changes(); changes == nil || len(changes) != 0 {
		t.Fatalf("empty plan changes = %#v, want non-nil empty slice", changes)
	}
	if err := plan.Apply(); err != nil {
		t.Fatalf("apply empty plan failed: %v", err)
	}
}

func TestBuildPlanIsSideEffectFreeAndOwnsReplacementBytes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	original := []byte("KEY=value  \n")
	mustWrite(t, path, original, 0o640)
	document := loadDocument(t, path, ".env")
	if err := os.Remove(path); err != nil {
		t.Fatalf("remove source after loading failed: %v", err)
	}

	plan, err := mutations.BuildPlan([]sourcedotenv.Document{document}, []lint.Rule{
		rules.NewTrailingWhitespace(),
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("planning touched removed source: stat error = %v", err)
	}
	changes := plan.Changes()
	if len(changes) != 1 {
		t.Fatalf("planned changes = %d, want 1", len(changes))
	}
	if changes[0].SourcePath != path || changes[0].DisplayPath != ".env" {
		t.Fatalf("planned provenance = %#v", changes[0])
	}
	if !bytes.Equal(changes[0].Original, original) {
		t.Fatalf("planned original = %q, want %q", changes[0].Original, original)
	}
	if want := []byte("KEY=value\n"); !bytes.Equal(changes[0].Replacement, want) {
		t.Fatalf("planned replacement = %q, want %q", changes[0].Replacement, want)
	}

	changes[0].Original[0] = 'X'
	changes[0].Replacement[0] = 'X'
	changes[0].SourcePath = "mutated"
	again := plan.Changes()
	if again[0].SourcePath != path {
		t.Fatalf("plan source path changed through accessor: %q", again[0].SourcePath)
	}
	if !bytes.Equal(again[0].Original, original) {
		t.Fatalf("plan original changed through accessor: %q", again[0].Original)
	}
	if want := []byte("KEY=value\n"); !bytes.Equal(again[0].Replacement, want) {
		t.Fatalf("plan replacement changed through accessor: %q", again[0].Replacement)
	}
}

func TestApplyOneChangedFilePreservesContentAndMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit preservation is not portable on Windows")
	}

	root := t.TempDir()
	path := filepath.Join(root, ".env")
	mustWrite(t, path, []byte("KEY=value  \n"), 0o601)
	document := loadDocument(t, path, ".env")
	if document.Mode.Perm() == 0o644 {
		t.Fatalf("fixture did not retain a non-default mode: %o", document.Mode.Perm())
	}

	plan, err := mutations.BuildPlan([]sourcedotenv.Document{document}, []lint.Rule{
		rules.NewTrailingWhitespace(),
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}
	if err := plan.Apply(); err != nil {
		t.Fatalf("apply plan failed: %v", err)
	}

	assertFileBytes(t, path, []byte("KEY=value\n"))
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat replaced file failed: %v", err)
	}
	if got, want := info.Mode().Perm(), document.Mode.Perm(); got != want {
		t.Fatalf("replaced file mode = %o, want %o", got, want)
	}
	assertNoAtomicTemporaryFiles(t, root)
}

func TestApplyMultipleChangesPreservesDocumentOrder(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, ".env.first")
	secondPath := filepath.Join(root, ".env.second")
	mustWrite(t, firstPath, []byte("FIRST=value  \n"), 0o644)
	mustWrite(t, secondPath, []byte("SECOND=value  \n"), 0o644)

	first := loadDocument(t, firstPath, ".env.first")
	second := loadDocument(t, secondPath, ".env.second")
	plan, err := mutations.BuildPlan([]sourcedotenv.Document{second, first}, []lint.Rule{
		rules.NewTrailingWhitespace(),
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}

	changes := plan.Changes()
	if len(changes) != 2 {
		t.Fatalf("planned changes = %d, want 2", len(changes))
	}
	if changes[0].SourcePath != secondPath || changes[1].SourcePath != firstPath {
		t.Fatalf("planned order = %#v, want second then first", changes)
	}

	if err := plan.Apply(); err != nil {
		t.Fatalf("apply plan failed: %v", err)
	}
	assertFileBytes(t, firstPath, []byte("FIRST=value\n"))
	assertFileBytes(t, secondPath, []byte("SECOND=value\n"))
	assertNoAtomicTemporaryFiles(t, root)
}

func TestBuildPlanHonorsSelectedFixableRules(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	original := []byte("KEY=value  \n\n\n")
	mustWrite(t, path, original, 0o644)
	document := loadDocument(t, path, ".env")

	plan, err := mutations.BuildPlan([]sourcedotenv.Document{document}, []lint.Rule{
		rules.NewTrailingWhitespace(),
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}
	if err := plan.Apply(); err != nil {
		t.Fatalf("apply plan failed: %v", err)
	}
	assertFileBytes(t, path, []byte("KEY=value\n\n\n"))
}

func TestBuildPlanOmitsUnfixableOnlySelection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	original := []byte("KEY=value  \n")
	mustWrite(t, path, original, 0o644)
	document := loadDocument(t, path, ".env")

	plan, err := mutations.BuildPlan([]sourcedotenv.Document{document}, []lint.Rule{plainRule{}})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}
	if changes := plan.Changes(); changes == nil || len(changes) != 0 {
		t.Fatalf("unfixable-only changes = %#v, want non-nil empty slice", changes)
	}
	if err := plan.Apply(); err != nil {
		t.Fatalf("apply unfixable-only plan failed: %v", err)
	}
	assertFileBytes(t, path, original)
}

func TestBuildPlanPreservesCanonicalFixOrder(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	original := []byte("KEY=valueX  \r\nZ=1\n")
	mustWrite(t, path, original, 0o644)
	document := loadDocument(t, path, ".env")

	selected := append(rules.All(), markerStripRule{})
	plan, err := mutations.BuildPlan([]sourcedotenv.Document{document}, selected)
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}
	changes := plan.Changes()
	if len(changes) != 1 {
		t.Fatalf("planned changes = %d, want 1", len(changes))
	}
	if want := []byte("KEY=valuelf\nZ=1\n"); !bytes.Equal(changes[0].Replacement, want) {
		t.Fatalf("canonical replacement = %q, want %q", changes[0].Replacement, want)
	}
}

func TestApplyPreservesCRLFForScopedFix(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".env")
	original := []byte("KEY=value  \r\nNEXT=1\r\n")
	mustWrite(t, path, original, 0o644)
	document := loadDocument(t, path, ".env")

	plan, err := mutations.BuildPlan([]sourcedotenv.Document{document}, []lint.Rule{
		rules.NewTrailingWhitespace(),
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}
	if err := plan.Apply(); err != nil {
		t.Fatalf("apply plan failed: %v", err)
	}
	assertFileBytes(t, path, []byte("KEY=value\r\nNEXT=1\r\n"))
}

func TestApplyPreservesFileSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file symlink behavior is not portable on Windows")
	}

	root := t.TempDir()
	target := filepath.Join(root, "target.env")
	alias := filepath.Join(root, "alias.env")
	mustWrite(t, target, []byte("KEY=value  \n"), 0o640)
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	document := loadDocument(t, alias, ".env")
	plan, err := mutations.BuildPlan([]sourcedotenv.Document{document}, []lint.Rule{
		rules.NewTrailingWhitespace(),
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}
	if err := plan.Apply(); err != nil {
		t.Fatalf("apply plan failed: %v", err)
	}

	info, err := os.Lstat(alias)
	if err != nil {
		t.Fatalf("lstat alias failed: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("alias mode = %v, want symlink", info.Mode())
	}
	linkedTarget, err := os.Readlink(alias)
	if err != nil {
		t.Fatalf("read alias target failed: %v", err)
	}
	if linkedTarget != target {
		t.Fatalf("alias target = %q, want %q", linkedTarget, target)
	}
	assertFileBytes(t, target, []byte("KEY=value\n"))
	assertNoAtomicTemporaryFiles(t, root)
}

func TestApplyRevalidatesEachDestinationBeforeReplacement(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file symlink behavior is not portable on Windows")
	}

	root := t.TempDir()
	target := filepath.Join(root, "target.env")
	alias := filepath.Join(root, "alias.env")
	original := []byte("KEY=value  \n")
	mustWrite(t, target, original, 0o644)
	if err := os.Symlink(target, alias); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	first := loadDocument(t, target, ".env.first")
	second := loadDocument(t, alias, ".env.second")
	plan, err := mutations.BuildPlan([]sourcedotenv.Document{first, second}, []lint.Rule{
		&sequencedFixRule{},
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}
	if changes := plan.Changes(); len(changes) != 2 {
		t.Fatalf("planned changes = %d, want 2", len(changes))
	}

	err = plan.Apply()
	if err == nil {
		t.Fatal("expected second destination to be rejected after the first replacement")
	}
	if !strings.Contains(err.Error(), ".env.second") {
		t.Fatalf("error = %v, want affected display path", err)
	}
	assertFileBytes(t, target, []byte("FIRST=changed\n"))
	info, err := os.Lstat(alias)
	if err != nil {
		t.Fatalf("lstat alias failed: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("alias mode = %v, want symlink", info.Mode())
	}
	assertNoAtomicTemporaryFiles(t, root)
}

func TestApplyRejectsStaleDestinationBeforeWritingAnyFile(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, ".env.first")
	secondPath := filepath.Join(root, ".env.second")
	firstOriginal := []byte("FIRST=value  \n")
	secondOriginal := []byte("SECOND=value  \n")
	mustWrite(t, firstPath, firstOriginal, 0o644)
	mustWrite(t, secondPath, secondOriginal, 0o644)

	first := loadDocument(t, firstPath, ".env.first")
	second := loadDocument(t, secondPath, ".env.second")
	plan, err := mutations.BuildPlan([]sourcedotenv.Document{first, second}, []lint.Rule{
		rules.NewTrailingWhitespace(),
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}

	stale := []byte("SECOND=changed\n")
	mustWrite(t, secondPath, stale, 0o644)
	err = plan.Apply()
	if err == nil {
		t.Fatal("expected stale destination error")
	}
	if !strings.Contains(err.Error(), ".env.second") {
		t.Fatalf("error = %v, want affected display path", err)
	}
	assertFileBytes(t, firstPath, firstOriginal)
	assertFileBytes(t, secondPath, stale)
	assertNoAtomicTemporaryFiles(t, root)
}

func TestApplyRejectsStalePermissionsBeforeWriting(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission-bit validation is not portable on Windows")
	}

	root := t.TempDir()
	path := filepath.Join(root, ".env")
	original := []byte("KEY=value  \n")
	mustWrite(t, path, original, 0o640)
	document := loadDocument(t, path, ".env")
	plan, err := mutations.BuildPlan([]sourcedotenv.Document{document}, []lint.Rule{
		rules.NewTrailingWhitespace(),
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}

	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatalf("change destination permissions failed: %v", err)
	}
	err = plan.Apply()
	if err == nil {
		t.Fatal("expected stale permission error")
	}
	if !strings.Contains(err.Error(), ".env") {
		t.Fatalf("error = %v, want affected path", err)
	}
	assertFileBytes(t, path, original)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat stale file failed: %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("stale file mode = %o, want %o", got, want)
	}
}

func TestApplyFailureLeavesDestinationIntact(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permission fixtures are not portable on Windows")
	}

	root := t.TempDir()
	path := filepath.Join(root, ".env")
	original := []byte("KEY=value  \n")
	mustWrite(t, path, original, 0o644)
	document := loadDocument(t, path, ".env")
	plan, err := mutations.BuildPlan([]sourcedotenv.Document{document}, []lint.Rule{
		rules.NewTrailingWhitespace(),
	})
	if err != nil {
		t.Fatalf("build plan failed: %v", err)
	}

	if err := os.Chmod(root, 0o500); err != nil {
		t.Fatalf("lock destination directory failed: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(root, 0o755); err != nil {
			t.Errorf("restore destination directory failed: %v", err)
		}
	})
	probe, probeErr := os.CreateTemp(root, "vaar-permission-probe-*")
	if probeErr == nil {
		_ = probe.Close()
		_ = os.Remove(probe.Name())
		t.Skip("directory permissions are not enforced by this test environment")
	}

	err = plan.Apply()
	if err == nil {
		t.Fatal("expected atomic replacement failure")
	}
	if !strings.Contains(err.Error(), ".env") {
		t.Fatalf("error = %v, want affected path", err)
	}
	assertFileBytes(t, path, original)
	assertNoAtomicTemporaryFiles(t, root)
}

func loadDocument(t *testing.T, path, displayPath string) sourcedotenv.Document {
	t.Helper()
	document, err := sourcedotenv.Load(path, displayPath)
	if err != nil {
		t.Fatalf("load document failed: %v", err)
	}
	return document
}

func mustWrite(t *testing.T, path string, data []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, data, mode); err != nil {
		t.Fatalf("write %q failed: %v", path, err)
	}
	if err := os.Chmod(path, mode.Perm()); err != nil {
		t.Fatalf("chmod %q failed: %v", path, err)
	}
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %q failed: %v", path, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("bytes in %q = %q, want %q", path, got, want)
	}
}

func assertNoAtomicTemporaryFiles(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read temporary directory failed: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "vaar-atomic-") {
			t.Fatalf("temporary file was not cleaned up: %q", entry.Name())
		}
	}
}
