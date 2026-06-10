package refs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
	"github.com/Kubonsang/unity-fileid-graph/internal/parser"
)

func TestExtractFindsScriptMaterialAndLocalRefs(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", "refs_prefab.prefab"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	parsed, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Extract(parsed, "prefab", "testdata/fixtures/refs_prefab.prefab")

	if result.Status != "OK" {
		t.Fatalf("expected OK, got %q issues=%+v", result.Status, result.Issues)
	}
	if len(result.References) != 6 {
		t.Fatalf("expected 6 refs, got %d: %+v", len(result.References), result.References)
	}
	assertRef(t, result.References[0], 1000, "m_Component[0].component", 4000, "", 0, false, false)
	assertRef(t, result.References[1], 1000, "m_Component[1].component", 11400000, "", 0, false, false)
	assertRef(t, result.References[2], 1000, "m_Component[2].component", 23000, "", 0, false, false)
	assertRef(t, result.References[3], 11400000, "m_Script", 11500000, "0123456789abcdef0123456789abcdef", 3, true, true)
	assertRef(t, result.References[4], 11400000, "m_Target", 4000, "", 0, false, false)
	assertRef(t, result.References[5], 23000, "m_Materials[0]", 2100000, "fedcba9876543210fedcba9876543210", 2, true, true)
}

func TestExtractKeepsCRLFFreeAnalysisView(t *testing.T) {
	input := []byte("--- !u!114 &11400000\r\nMonoBehaviour:\r\n  m_Script: {fileID: 11500000, guid: 0123456789abcdef0123456789abcdef, type: 3}\r\n")
	parsed, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Extract(parsed, "prefab", "crlf.prefab")

	if len(result.References) != 1 {
		t.Fatalf("expected 1 ref, got %d", len(result.References))
	}
	if result.References[0].Field != "m_Script" {
		t.Fatalf("unexpected field %q", result.References[0].Field)
	}
}

func TestExtractWarnsOnOverflowingInlinePPtr(t *testing.T) {
	input := []byte("--- !u!114 &11400000\nMonoBehaviour:\n  m_Script: {fileID: 999999999999999999999999999999, guid: 0123456789abcdef0123456789abcdef, type: 3}\n")
	parsed, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Extract(parsed, "prefab", "overflow.prefab")

	if result.Status != core.RefsStatusWarn {
		t.Fatalf("expected WARN, got %q", result.Status)
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d: %+v", len(result.Issues), result.Issues)
	}
	if result.Issues[0].Code != core.IssueUnknownFieldShape {
		t.Fatalf("expected UNKNOWN_FIELD_SHAPE, got %+v", result.Issues[0])
	}
}

func TestExtractSkipsTransformHierarchyStructuralRefs(t *testing.T) {
	input := []byte("--- !u!4 &4000\nTransform:\n  m_GameObject: {fileID: 1000}\n  m_Father: {fileID: 3999}\n  m_Children:\n  - {fileID: 4001}\n")
	parsed, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	result := Extract(parsed, "prefab", "transform.prefab")

	if len(result.References) != 0 {
		t.Fatalf("expected structural Transform refs to be skipped, got %+v", result.References)
	}
}

func assertRef(t *testing.T, got core.Reference, blockID int64, field string, fileID int64, guid string, typeValue int, hasGUID bool, hasType bool) {
	t.Helper()
	if got.BlockFileID != blockID || got.Field != field || got.FileID != fileID || got.GUID != guid || got.Type != typeValue || got.HasGUID != hasGUID || got.HasType != hasType {
		t.Fatalf("unexpected ref: %+v", got)
	}
}
