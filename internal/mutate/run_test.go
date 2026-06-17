package mutate

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/pkg/core"
)

func TestRunSetMutatesExistingFieldAndCreatesBackup(t *testing.T) {
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_prefab.prefab")
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "set_prefab.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	result, err := RunSet(core.SetOptions{
		InputPath: target,
		FileID:    1000,
		Field:     "m_IsActive",
		Value:     "0",
	})
	if err != nil {
		t.Fatalf("RunSet returned error: %v", err)
	}
	if result.Status != core.MutationStatusOK {
		t.Fatalf("expected %q, got %q", core.MutationStatusOK, result.Status)
	}
	mutated, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read mutated file: %v", err)
	}
	if !strings.Contains(string(mutated), "m_IsActive: 0") {
		t.Fatalf("expected field update, got %q", string(mutated))
	}
	if _, err := os.Stat(target + ".bak"); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
}

// TestRunSetReportsPreCheckSkippedForStrippedChild verifies skip visibility
// propagates to the write path: a SET whose pre_check skipped a stripped/unmodeled
// transform-symmetry link reports that on the result, so a write is never read as
// "fully symmetry-checked" when it was not.
func TestRunSetReportsPreCheckSkippedForStrippedChild(t *testing.T) {
	source := filepath.Join("..", "..", "testdata", "fixtures", "transform_children_f3_stripped_child.prefab")
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "stripped.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	result, err := RunSet(core.SetOptions{
		InputPath: target,
		FileID:    1000,
		Field:     "m_IsActive",
		Value:     "0",
	})
	if err != nil {
		t.Fatalf("RunSet returned error: %v", err)
	}
	if result.Status != core.MutationStatusOK {
		t.Fatalf("expected OK, got %q", result.Status)
	}
	if result.PreCheckSkippedLinks != 1 || result.PreCheckSkippedStripped != 1 || result.PreCheckSkippedUnmodeledClass != 0 {
		t.Fatalf("write-path skip visibility wrong: links=%d stripped=%d unmodeled=%d",
			result.PreCheckSkippedLinks, result.PreCheckSkippedStripped, result.PreCheckSkippedUnmodeledClass)
	}
}

func TestRunSetReturnsWarnWhenPreAndFinalChecksWarn(t *testing.T) {
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_material.mat")
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "set_material.mat")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	result, err := RunSet(core.SetOptions{
		InputPath: target,
		FileID:    2100000,
		Field:     "m_Name",
		Value:     "Helmet",
	})
	if err != nil {
		t.Fatalf("RunSet returned error: %v", err)
	}
	if result.Status != core.MutationStatusWarn {
		t.Fatalf("expected %q, got %q", core.MutationStatusWarn, result.Status)
	}
}

func TestRunSetReturnsBlockedForMonoBehaviour(t *testing.T) {
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_monobehaviour_blocked.prefab")
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "set_monobehaviour_blocked.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	result, err := RunSet(core.SetOptions{
		InputPath: target,
		FileID:    11400000,
		Field:     "m_Enabled",
		Value:     "0",
	})
	if err != nil {
		t.Fatalf("RunSet returned error: %v", err)
	}
	if result.Status != core.MutationStatusBlocked {
		t.Fatalf("expected %q, got %q", core.MutationStatusBlocked, result.Status)
	}
}

func TestRunSetReturnsBlockedForDuplicateFileID(t *testing.T) {
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_duplicate_fileid_blocked.prefab")
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "set_duplicate_fileid_blocked.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	result, err := RunSet(core.SetOptions{
		InputPath: target,
		FileID:    1000,
		Field:     "m_IsActive",
		Value:     "0",
	})
	if err != nil {
		t.Fatalf("RunSet returned error: %v", err)
	}
	if result.Status != core.MutationStatusBlocked || result.Code != core.MutationCodeDuplicateFileID {
		t.Fatalf("expected duplicate blocked result, got %+v", result)
	}
}

func TestRunSetPreservesUnrelatedBlockBytes(t *testing.T) {
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_prefab.prefab")
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "set_prefab.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	result, err := RunSet(core.SetOptions{
		InputPath: target,
		FileID:    1000,
		Field:     "m_IsActive",
		Value:     "0",
	})
	if err != nil {
		t.Fatalf("RunSet returned error: %v", err)
	}
	if result.Status != core.MutationStatusOK {
		t.Fatalf("expected OK, got %+v", result)
	}
	mutated, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read mutated: %v", err)
	}
	if !strings.Contains(string(mutated), "m_RootOrder: 0") {
		t.Fatalf("expected unrelated transform line to stay unchanged, got %q", string(mutated))
	}
}

func TestRunSetCleansTempFileOnBlockedPath(t *testing.T) {
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_monobehaviour_blocked.prefab")
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "set_monobehaviour_blocked.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	if _, err := RunSet(core.SetOptions{
		InputPath: target,
		FileID:    11400000,
		Field:     "m_Enabled",
		Value:     "0",
	}); err != nil {
		t.Fatalf("RunSet returned error: %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(tempDir, "set_monobehaviour_blocked.prefab.tmp-*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no leftover temp files, got %v", matches)
	}
}

func TestReplaceWithBackupRestoresOriginalOnRenameFailure(t *testing.T) {
	tempDir := t.TempDir()
	originalPath := filepath.Join(tempDir, "sample.prefab")
	tempPath := filepath.Join(tempDir, "sample.prefab.tmp")
	if err := os.WriteFile(originalPath, []byte("original"), 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}
	if err := os.WriteFile(tempPath, []byte("mutated"), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}

	ops := fileOps{
		Rename: func(oldpath, newpath string) error {
			if oldpath == tempPath && newpath == originalPath {
				return errors.New("boom")
			}
			return os.Rename(oldpath, newpath)
		},
		Remove: os.Remove,
	}

	_, err := replaceWithBackup(originalPath, tempPath, ops)
	if err == nil {
		t.Fatalf("expected replace failure")
	}
	restored, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("read restored original: %v", err)
	}
	if string(restored) != "original" {
		t.Fatalf("expected original content to be restored, got %q", string(restored))
	}
}

func TestCompleteWritePipelineMarksRestoreFailedWhenBackupRestoreFails(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "sample.prefab")

	original := []byte("--- !u!1 &1000\nGameObject:\n  m_Name: Original\n")
	mutated := []byte("--- !u!1 &1000\nGameObject:\n  m_Name: Mutated\n")
	corrupted := []byte("--- !u!1 &1000\nGameObject:\n  m_Name: Corrupted\n")

	if err := os.WriteFile(target, original, 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}

	ops := fileOps{
		Rename: func(oldpath, newpath string) error {
			if strings.HasSuffix(oldpath, ".bak") || strings.Contains(oldpath, ".bak.") {
				return errors.New("restore failed")
			}
			return os.Rename(oldpath, newpath)
		},
		Remove: os.Remove,
	}

	pipeline, err := completeWritePipeline(target, mutated, ops, writePipelineOptions{
		RestoreOnFinalCheckError: true,
		CheckBytes: func(phase writePipelineCheckPhase, got []byte) (string, error) {
			switch phase {
			case writePipelineCheckTemp:
				return core.CheckStatusOK, nil
			case writePipelineCheckFinal:
				if string(got) != string(corrupted) {
					t.Fatalf("expected corrupted final bytes")
				}
				return core.CheckStatusError, nil
			default:
				t.Fatalf("unexpected phase: %q", phase)
				return "", nil
			}
		},
		AfterReplace: func(path string) error {
			return os.WriteFile(path, corrupted, 0o644)
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pipeline.Restored {
		t.Fatalf("expected restored=false")
	}
	if !pipeline.RestoreFail {
		t.Fatalf("expected restore_failed=true")
	}
}

func TestRunSetRestoresOriginalOnFinalCheckError(t *testing.T) {
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_prefab.prefab")
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "set_prefab.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	corrupted := []byte("--- !u!1 &1000\nGameObject:\n  m_Name: Corrupted\n")

	result, err := runSetWithFileOps(core.SetOptions{
		InputPath: target,
		FileID:    1000,
		Field:     "m_IsActive",
		Value:     "0",
	}, defaultFileOps(), writePipelineOptions{
		RestoreOnFinalCheckError: true,
		AfterReplace: func(path string) error {
			return os.WriteFile(path, corrupted, 0o644)
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != core.MutationStatusError {
		t.Fatalf("expected ERROR, got %q", result.Status)
	}
	if result.Code != core.MutationCodeFinalCheckError {
		t.Fatalf("expected FINAL_CHECK_ERROR, got %q", result.Code)
	}
	if result.Message != "restored=true" {
		t.Fatalf("expected restored=true message, got %q", result.Message)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read restored target: %v", err)
	}
	if string(got) != string(input) {
		t.Fatalf("expected original bytes restored")
	}
}
