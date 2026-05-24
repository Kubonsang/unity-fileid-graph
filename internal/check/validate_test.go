package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
	"github.com/Kubonsang/unity-fileid-graph/internal/graph"
	"github.com/Kubonsang/unity-fileid-graph/internal/parser"
)

func TestRunReturnsOKForHealthyFixture(t *testing.T) {
	graphResult := buildFixtureGraph(t, "check_ok.prefab")

	result := Run(graphResult)

	if result.Status != core.CheckStatusOK {
		t.Fatalf("expected status %q, got %q", core.CheckStatusOK, result.Status)
	}
	if result.BlockCount != 2 || result.GameObjectCount != 1 || result.ComponentCount != 1 || result.TransformCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
}

func TestRunNilGraphReturnsInitializedOKResult(t *testing.T) {
	result := Run(nil)

	if result == nil {
		t.Fatalf("expected initialized result")
	}
	if result.Status != core.CheckStatusOK {
		t.Fatalf("expected status %q, got %q", core.CheckStatusOK, result.Status)
	}
	if result.BlockCount != 0 || result.GameObjectCount != 0 || result.ComponentCount != 0 || result.TransformCount != 0 {
		t.Fatalf("expected zero counts, got %+v", result)
	}
	if result.Errors == nil || result.Warnings == nil {
		t.Fatalf("expected initialized slices, got %+v", result)
	}
	if len(result.Errors) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("expected empty findings, got errors=%v warnings=%v", result.Errors, result.Warnings)
	}
}

func TestRunDetectsDuplicateFileID(t *testing.T) {
	graphResult := buildFixtureGraph(t, "check_duplicate_fileid.prefab")

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if result.BlockCount != 4 || result.GameObjectCount != 2 || result.ComponentCount != 2 || result.TransformCount != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckDuplicateFileID || result.Errors[0].FileID != 900 || result.Errors[0].Reason != "duplicate_block_headers" {
		t.Fatalf("unexpected first error: %+v", result.Errors[0])
	}
	if result.Errors[1].Code != core.CheckDuplicateFileID || result.Errors[1].FileID != 1000 || result.Errors[1].Reason != "duplicate_block_headers" {
		t.Fatalf("unexpected second error: %+v", result.Errors[1])
	}
}

func TestRunDetectsMissingComponentBlock(t *testing.T) {
	graphResult := buildFixtureGraph(t, "check_missing_component.prefab")

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if result.BlockCount != 3 || result.GameObjectCount != 2 || result.ComponentCount != 1 || result.TransformCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckMissingComponentBlock || result.Errors[0].GameObjectID != 1000 || result.Errors[0].ComponentID != 11400000 || result.Errors[0].Reason != "missing_component_block" {
		t.Fatalf("unexpected first error: %+v", result.Errors[0])
	}
	if result.Errors[1].Code != core.CheckMissingComponentBlock || result.Errors[1].GameObjectID != 2000 || result.Errors[1].ComponentID != 11400001 || result.Errors[1].Reason != "missing_component_block" {
		t.Fatalf("unexpected second error: %+v", result.Errors[1])
	}
}

func TestRunDetectsMissingGameObjectBlock(t *testing.T) {
	graphResult := buildFixtureGraph(t, "check_missing_gameobject.prefab")

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if result.BlockCount != 2 || result.GameObjectCount != 0 || result.ComponentCount != 2 || result.TransformCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckMissingGameObjectBlock || result.Errors[0].GameObjectID != 2000 || result.Errors[0].ComponentID != 500 || result.Errors[0].Reason != "missing_gameobject_block" {
		t.Fatalf("unexpected first error: %+v", result.Errors[0])
	}
	if result.Errors[1].Code != core.CheckMissingGameObjectBlock || result.Errors[1].GameObjectID != 1000 || result.Errors[1].ComponentID != 11400000 || result.Errors[1].Reason != "missing_gameobject_block" {
		t.Fatalf("unexpected second error: %+v", result.Errors[1])
	}
}

func buildFixtureGraph(t *testing.T, name string) *core.Graph {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "fixtures", name)
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", name, err)
	}

	parsed, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse fixture %q: %v", name, err)
	}

	graphResult, err := graph.Build(parsed)
	if err != nil {
		t.Fatalf("build fixture %q: %v", name, err)
	}

	return graphResult
}
