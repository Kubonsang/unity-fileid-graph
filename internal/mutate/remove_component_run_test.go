package mutate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

func TestRunRemoveComponentRemovesBoxColliderAndPreservesTransform(t *testing.T) {
	target := copyFixture(t, "remove_component_ok.prefab")

	result, err := RunRemoveComponent(core.RemoveComponentOptions{
		InputPath:    target,
		FileID:       65000,
		Experimental: true,
		Write:        true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != core.MutationStatusExperimental {
		t.Fatalf("expected EXPERIMENTAL, got %q", result.Status)
	}
	bytes, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if strings.Contains(string(bytes), "&65000") {
		t.Fatalf("expected target block to be removed")
	}
	if strings.Contains(string(bytes), "component: {fileID: 65000}") {
		t.Fatalf("expected GameObject component ref to be removed")
	}
	if !strings.Contains(string(bytes), "&4000") {
		t.Fatalf("expected Transform block to remain")
	}
}

func TestRunRemoveComponentBlocksTransformTargets(t *testing.T) {
	result, err := RunRemoveComponent(core.RemoveComponentOptions{
		InputPath:    fixturePath("remove_component_transform_blocked.prefab"),
		FileID:       4000,
		Experimental: true,
		Write:        true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != core.MutationStatusBlocked || result.Code != core.MutationCodeTransformRemoveBlocked {
		t.Fatalf("expected transform blocked result, got status=%q code=%q", result.Status, result.Code)
	}
}

func TestRunRemoveComponentBlocksOwnerMismatch(t *testing.T) {
	result, err := RunRemoveComponent(core.RemoveComponentOptions{
		InputPath:    fixturePath("remove_component_owner_mismatch_blocked.prefab"),
		FileID:       65000,
		Experimental: true,
		Write:        true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != core.MutationStatusBlocked || result.Code != core.MutationCodeComponentOwnerMismatch {
		t.Fatalf("expected owner mismatch blocked result, got status=%q code=%q", result.Status, result.Code)
	}
}

func TestCompleteWritePipelineRestoresBackupOnFinalCheckError(t *testing.T) {
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, "sample.prefab")

	original := []byte("--- !u!1 &1000\nGameObject:\n  m_Name: Original\n")
	mutated := []byte("--- !u!1 &1000\nGameObject:\n  m_Name: Mutated\n")

	if err := os.WriteFile(target, original, 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}

	checkCount := 0
	pipeline, err := completeWritePipeline(target, mutated, defaultFileOps(), writePipelineOptions{
		RestoreOnFinalCheckError: true,
		CheckBytes: func(_ []byte) (string, error) {
			checkCount++
			if checkCount == 1 {
				return core.CheckStatusOK, nil
			}
			return core.CheckStatusError, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !pipeline.Restored {
		t.Fatalf("expected restored=true")
	}
	if pipeline.RestoreFail {
		t.Fatalf("expected restore_failed=false")
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(got) != string(original) {
		t.Fatalf("expected original restored, got %q", string(got))
	}
}

func fixturePath(name string) string {
	return filepath.Join("..", "..", "testdata", "fixtures", name)
}

func copyFixture(t *testing.T, name string) string {
	t.Helper()
	source := fixturePath(name)
	tempDir := t.TempDir()
	target := filepath.Join(tempDir, name)
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	return target
}
