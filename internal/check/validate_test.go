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
	if result.BlockCount != 6 || result.GameObjectCount != 2 || result.ComponentCount != 4 || result.TransformCount != 2 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckDuplicateFileID || result.Errors[0].FileID != 900 || result.Errors[0].DuplicateCount != 2 || result.Errors[0].Reason != "duplicate_block_headers" {
		t.Fatalf("unexpected first error: %+v", result.Errors[0])
	}
	if result.Errors[1].Code != core.CheckDuplicateFileID || result.Errors[1].FileID != 1000 || result.Errors[1].DuplicateCount != 2 || result.Errors[1].Reason != "duplicate_block_headers" {
		t.Fatalf("unexpected second error: %+v", result.Errors[1])
	}
}

func TestRunDetectsMissingComponentBlock(t *testing.T) {
	graphResult := buildFixtureGraph(t, "check_missing_component.prefab")

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if result.BlockCount != 4 || result.GameObjectCount != 2 || result.ComponentCount != 2 || result.TransformCount != 2 {
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

func TestRunDetectsComponentReferenceToNonComponentBlock(t *testing.T) {
	graphResult := buildGraphFromString(t,
		"--- !u!1 &1000\n"+
			"GameObject:\n"+
			"  m_Component:\n"+
			"  - component: {fileID: 4000}\n"+
			"  - component: {fileID: 2000}\n"+
			"  m_Name: Root\n"+
			"--- !u!4 &4000\n"+
			"Transform:\n"+
			"  m_GameObject: {fileID: 1000}\n"+
			"  m_Father: {fileID: 0}\n"+
			"  m_Children: []\n"+
			"--- !u!1 &2000\n"+
			"GameObject:\n"+
			"  m_Component:\n"+
			"  - component: {fileID: 5000}\n"+
			"  m_Name: NotAComponent\n"+
			"--- !u!4 &5000\n"+
			"Transform:\n"+
			"  m_GameObject: {fileID: 2000}\n"+
			"  m_Father: {fileID: 0}\n"+
			"  m_Children: []\n",
	)

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(result.Errors), result.Errors)
	}
	if result.Errors[0].Code != core.CheckMissingComponentBlock || result.Errors[0].GameObjectID != 1000 || result.Errors[0].ComponentID != 2000 || result.Errors[0].Reason != "referenced_block_is_not_component" {
		t.Fatalf("unexpected error: %+v", result.Errors[0])
	}
}

func TestRunDetectsComponentReferenceToMaterialBlock(t *testing.T) {
	graphResult := buildGraphFromString(t,
		"--- !u!1 &1000\n"+
			"GameObject:\n"+
			"  m_Component:\n"+
			"  - component: {fileID: 2100000}\n"+
			"  m_Name: Root\n"+
			"--- !u!21 &2100000\n"+
			"Material:\n"+
			"  m_Name: NotAComponent\n",
	)

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d: %+v", len(result.Errors), result.Errors)
	}
	if result.Errors[0].Code != core.CheckMissingComponentBlock || result.Errors[0].GameObjectID != 1000 || result.Errors[0].ComponentID != 2100000 || result.Errors[0].Reason != "referenced_block_is_not_component" {
		t.Fatalf("unexpected error: %+v", result.Errors[0])
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

func TestRunDetectsGameObjectComponentBackrefMismatch(t *testing.T) {
	graphResult := buildFixtureGraph(t, "check_backref_mismatch.prefab")

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if result.BlockCount != 5 || result.GameObjectCount != 2 || result.ComponentCount != 3 || result.TransformCount != 2 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckGoComponentBackrefMismatch || result.Errors[0].GameObjectID != 1000 || result.Errors[0].ComponentID != 11400000 || result.Errors[0].Reason != "component_points_to_other_gameobject" {
		t.Fatalf("unexpected error: %+v", result.Errors[0])
	}
}

func TestRunDetectsTransformParentChildMismatch(t *testing.T) {
	graphResult := buildFixtureGraph(t, "check_transform_mismatch.prefab")

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if result.BlockCount != 6 || result.GameObjectCount != 3 || result.ComponentCount != 3 || result.TransformCount != 3 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Errors) != 3 {
		t.Fatalf("expected 3 errors, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckTransformParentChildMismatch || result.Errors[0].TransformID != 4000 || result.Errors[0].ParentID != 9999 || result.Errors[0].Reason != "missing_parent_transform" {
		t.Fatalf("unexpected first error: %+v", result.Errors[0])
	}
	if result.Errors[1].Code != core.CheckTransformParentChildMismatch || result.Errors[1].TransformID != 4000 || result.Errors[1].ChildID != 4001 || result.Errors[1].Reason != "child_father_mismatch" {
		t.Fatalf("unexpected second error: %+v", result.Errors[1])
	}
	if result.Errors[2].Code != core.CheckTransformParentChildMismatch || result.Errors[2].TransformID != 4000 || result.Errors[2].ChildID != 8888 || result.Errors[2].Reason != "missing_child_transform" {
		t.Fatalf("unexpected third error: %+v", result.Errors[2])
	}
}

func TestRunDetectsMissingTransformComponentOnGameObject(t *testing.T) {
	graphResult := buildFixtureGraph(t, "check_missing_transform.prefab")

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if result.BlockCount != 2 || result.GameObjectCount != 1 || result.ComponentCount != 1 || result.TransformCount != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckMissingTransformComponent || result.Errors[0].GameObjectID != 1000 || result.Errors[0].Reason != "missing_transform_component" {
		t.Fatalf("unexpected error: %+v", result.Errors[0])
	}
}

func TestRunDetectsSuspiciousMonoBehaviourScript(t *testing.T) {
	graphResult := buildFixtureGraph(t, "check_suspicious_script.prefab")

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if result.BlockCount != 3 || result.GameObjectCount != 1 || result.ComponentCount != 2 || result.TransformCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %v", result.Warnings)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckSuspiciousMonoBehaviourScript || result.Errors[0].ComponentID != 11400000 || result.Errors[0].Reason != "missing_script_metadata" {
		t.Fatalf("unexpected error: %+v", result.Errors[0])
	}
}

func TestRunDetectsSuspiciousMonoBehaviourScriptWhenMetadataShapeIsMalformed(t *testing.T) {
	graphResult := &core.Graph{
		Blocks:      []*core.Block{{FileID: 11400000, ClassID: 114}},
		BlocksByID:  map[int64][]*core.Block{11400000: []*core.Block{{FileID: 11400000, ClassID: 114}}},
		ObjectsByID: map[int64][]*core.UnityObject{},
		GameObjects: map[int64]*core.GameObjectNode{},
		Components: map[int64]*core.ComponentNode{
			11400000: {
				FileID:        11400000,
				ClassID:       114,
				TypeName:      "MonoBehaviour",
				HasGameObject: false,
				Script:        nil,
			},
		},
		Transforms: map[int64]*core.TransformNode{},
		Issues: []core.Issue{
			{Code: core.IssueUnknownFieldShape, FileID: 11400000, Message: "unsupported MonoBehaviour.m_Script shape"},
		},
	}

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckSuspiciousMonoBehaviourScript || result.Errors[0].ComponentID != 11400000 || result.Errors[0].Reason != "missing_script_metadata" {
		t.Fatalf("unexpected suspicious-script error: %+v", result.Errors[0])
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning passthrough, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Code != core.IssueUnknownFieldShape || result.Warnings[0].FileID != 11400000 {
		t.Fatalf("unexpected warning passthrough: %+v", result.Warnings[0])
	}
}

func TestRunReturnsWarnStatusWhenGraphOnlyHasWarnings(t *testing.T) {
	graphResult := &core.Graph{
		Blocks:      []*core.Block{},
		BlocksByID:  map[int64][]*core.Block{},
		ObjectsByID: map[int64][]*core.UnityObject{},
		GameObjects: map[int64]*core.GameObjectNode{},
		Components:  map[int64]*core.ComponentNode{},
		Transforms:  map[int64]*core.TransformNode{},
		Issues: []core.Issue{
			{Code: core.IssueTabIndent, FileID: 1000, Message: "tab indentation is unsupported in v0.2 field scanning"},
		},
	}

	result := Run(graphResult)

	if result.Status != core.CheckStatusWarn {
		t.Fatalf("expected status %q, got %q", core.CheckStatusWarn, result.Status)
	}
	if result.BlockCount != 0 || result.GameObjectCount != 0 || result.ComponentCount != 0 || result.TransformCount != 0 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Code != core.IssueTabIndent || result.Warnings[0].FileID != 1000 || result.Warnings[0].Message != "tab indentation is unsupported in v0.2 field scanning" {
		t.Fatalf("unexpected warning: %+v", result.Warnings[0])
	}
}

func TestRunSkipsBackrefAndMissingTransformChecksWhenReferencedComponentHasGraphIssue(t *testing.T) {
	graphResult := &core.Graph{
		GameObjects: map[int64]*core.GameObjectNode{
			1000: {FileID: 1000, Components: []int64{11400000}},
		},
		Components: map[int64]*core.ComponentNode{
			11400000: {
				FileID:        11400000,
				ClassID:       4,
				TypeName:      "Transform",
				HasGameObject: false,
			},
		},
		Transforms: map[int64]*core.TransformNode{},
		Issues: []core.Issue{
			{Code: core.IssueUnknownFieldShape, FileID: 11400000, Message: "unsupported Component.m_GameObject shape"},
		},
	}

	result := Run(graphResult)

	if result.Status != core.CheckStatusWarn {
		t.Fatalf("expected status %q, got %q", core.CheckStatusWarn, result.Status)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no false-positive errors, got %v", result.Errors)
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Code != core.IssueUnknownFieldShape {
		t.Fatalf("expected passthrough warning only, got %v", result.Warnings)
	}
}

func TestRunSkipsParentChildMismatchWhenChildTransformHasGraphIssue(t *testing.T) {
	graphResult := &core.Graph{
		Transforms: map[int64]*core.TransformNode{
			4000: {FileID: 4000, Children: []int64{4001}},
			4001: {FileID: 4001, Father: 0},
		},
		Issues: []core.Issue{
			{Code: core.IssueUnknownFieldShape, FileID: 4001, Message: "unsupported Transform.m_Father shape"},
		},
	}

	result := Run(graphResult)

	if result.Status != core.CheckStatusWarn {
		t.Fatalf("expected status %q, got %q", core.CheckStatusWarn, result.Status)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no false-positive errors, got %v", result.Errors)
	}
	if len(result.Warnings) != 1 || result.Warnings[0].Code != core.IssueUnknownFieldShape {
		t.Fatalf("expected passthrough warning only, got %v", result.Warnings)
	}
}

func TestRunDoesNotTreatUnsupportedPresentComponentBlockAsMissing(t *testing.T) {
	graphResult := &core.Graph{
		Blocks: []*core.Block{
			{FileID: 1000, ClassID: 1},
			{FileID: 4000, ClassID: 4},
			{FileID: 23000, ClassID: 23},
		},
		BlocksByID: map[int64][]*core.Block{
			1000:  {{FileID: 1000, ClassID: 1}},
			4000:  {{FileID: 4000, ClassID: 4}},
			23000: {{FileID: 23000, ClassID: 23}},
		},
		ObjectsByID: map[int64][]*core.UnityObject{
			1000:  {{FileID: 1000, ClassID: 1, TypeName: "GameObject"}},
			4000:  {{FileID: 4000, ClassID: 4, TypeName: "Transform"}},
			23000: {{FileID: 23000, ClassID: 23, TypeName: "MeshRenderer"}},
		},
		GameObjects: map[int64]*core.GameObjectNode{
			1000: {FileID: 1000, Name: "Player", Components: []int64{23000, 4000}, Transform: 4000},
		},
		Components: map[int64]*core.ComponentNode{
			4000: {FileID: 4000, ClassID: 4, TypeName: "Transform", HasGameObject: true, GameObject: 1000},
		},
		Transforms: map[int64]*core.TransformNode{
			4000: {FileID: 4000, GameObject: 1000},
		},
	}

	result := Run(graphResult)

	if result.Status != core.CheckStatusOK {
		t.Fatalf("expected status %q, got %q (errors=%v warnings=%v)", core.CheckStatusOK, result.Status, result.Errors, result.Warnings)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors for unsupported-but-present component block, got %v", result.Errors)
	}
}

func TestRunDetectsMissingReverseParentChildLink(t *testing.T) {
	graphResult := &core.Graph{
		Transforms: map[int64]*core.TransformNode{
			4000: {FileID: 4000, Children: []int64{}},
			4001: {FileID: 4001, Father: 4000},
		},
	}

	result := Run(graphResult)

	if result.Status != core.CheckStatusError {
		t.Fatalf("expected status %q, got %q", core.CheckStatusError, result.Status)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result.Errors))
	}
	if result.Errors[0].Code != core.CheckTransformParentChildMismatch || result.Errors[0].TransformID != 4001 || result.Errors[0].ParentID != 4000 || result.Errors[0].Reason != "missing_from_parent_children" {
		t.Fatalf("unexpected error: %+v", result.Errors[0])
	}
}

func buildFixtureGraph(t *testing.T, name string) *core.Graph {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "fixtures", name)
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", name, err)
	}
	return buildGraphFromBytes(t, input)
}

func buildGraphFromString(t *testing.T, input string) *core.Graph {
	t.Helper()
	return buildGraphFromBytes(t, []byte(input))
}

func buildGraphFromBytes(t *testing.T, input []byte) *core.Graph {
	t.Helper()

	parsed, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse input: %v", err)
	}

	graphResult, err := graph.Build(parsed)
	if err != nil {
		t.Fatalf("build input: %v", err)
	}

	return graphResult
}
