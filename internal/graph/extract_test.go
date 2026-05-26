package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

func TestExtractGameObjectPreservesComponentOrder(t *testing.T) {
	body := "" +
		"GameObject:\n" +
		"  m_Name: Player\n" +
		"  m_Component:\n" +
		"  - component: {fileID: 11400000}\n" +
		"  - component: {fileID: 4000}\n"

	got, issues := extractGameObject(1000, body)

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if len(got.Components) != 2 {
		t.Fatalf("expected 2 components, got %v", got.Components)
	}
	if got.Components[0] != 11400000 || got.Components[1] != 4000 {
		t.Fatalf("expected source order preserved, got %v", got.Components)
	}
}

func TestExtractGameObjectUnknownComponentShapeReturnsIssue(t *testing.T) {
	body := string(loadGraphFixture(t, "unknown_component_shape.prefab"))

	got, issues := extractGameObject(1000, body)

	if len(got.Components) != 0 {
		t.Fatalf("expected no extracted components, got %v", got.Components)
	}
	if len(issues) != 1 || issues[0].Code != core.IssueUnknownFieldShape {
		t.Fatalf("expected UNKNOWN_FIELD_SHAPE issue, got %v", issues)
	}
}

func TestExtractTransformChildrenUsesChildListParser(t *testing.T) {
	body := "" +
		"Transform:\n" +
		"  m_GameObject: {fileID: 1000}\n" +
		"  m_Father: {fileID: 0}\n" +
		"  m_Children:\n" +
		"    - {fileID: 4001}\n" +
		"    - {fileID: 4002}\n"

	component, transform, issues := extractTransform(4000, body)

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if !component.HasGameObject || component.GameObject != 1000 {
		t.Fatalf("expected game object 1000, got %d", component.GameObject)
	}
	if len(transform.Children) != 2 || transform.Children[0] != 4001 || transform.Children[1] != 4002 {
		t.Fatalf("unexpected children: %v", transform.Children)
	}
}

func TestExtractMonoBehaviourUnknownScriptShapeReturnsIssue(t *testing.T) {
	body := string(loadGraphFixture(t, "unknown_script_shape.prefab"))

	component, issues := extractMonoBehaviour(11400000, body)

	if component.Script != nil {
		t.Fatalf("expected no script metadata")
	}
	if len(issues) != 1 || issues[0].Code != core.IssueUnknownFieldShape {
		t.Fatalf("expected UNKNOWN_FIELD_SHAPE issue, got %v", issues)
	}
}

func TestExtractMonoBehaviourParsesScriptMetadata(t *testing.T) {
	body := "" +
		"MonoBehaviour:\n" +
		"  m_GameObject: {fileID: 1000}\n" +
		"  m_Script: {fileID: 11500000, guid: 0123456789abcdef0123456789abcdef, type: 3}\n"

	component, issues := extractMonoBehaviour(11400000, body)

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if component.Script == nil {
		t.Fatalf("expected script metadata to be extracted")
	}
	if component.Script.FileID != 11500000 {
		t.Fatalf("expected script file id 11500000, got %d", component.Script.FileID)
	}
	if component.Script.GUID != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("unexpected script guid: %q", component.Script.GUID)
	}
	if component.Script.Type != 3 {
		t.Fatalf("expected script type 3, got %d", component.Script.Type)
	}
}

func TestExtractMonoBehaviourPopulatesScriptRefFromInlineMetadata(t *testing.T) {
	body := "" +
		"MonoBehaviour:\n" +
		"  m_GameObject: {fileID: 1000}\n" +
		"  m_Script: {fileID: 11500000, guid: 0123456789abcdef0123456789abcdef, type: 3}\n"

	component, issues := extractMonoBehaviour(11400000, body)

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if component.Script == nil {
		t.Fatalf("expected script metadata")
	}
	if component.Script.FileID != 11500000 {
		t.Fatalf("expected script fileID 11500000, got %d", component.Script.FileID)
	}
	if component.Script.GUID != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("expected script GUID populated, got %q", component.Script.GUID)
	}
	if component.Script.Type != 3 {
		t.Fatalf("expected script type 3, got %d", component.Script.Type)
	}
}

func TestExtractTransformUnknownChildrenShapeReturnsIssue(t *testing.T) {
	body := "" +
		"Transform:\n" +
		"  m_GameObject: {fileID: 1000}\n" +
		"  m_Children:\n" +
		"    - child:\n" +
		"        nested: unexpected\n"

	_, transform, issues := extractTransform(4000, body)

	if len(transform.Children) != 0 {
		t.Fatalf("expected no extracted children, got %v", transform.Children)
	}
	if len(issues) != 1 || issues[0].Code != core.IssueUnknownFieldShape {
		t.Fatalf("expected UNKNOWN_FIELD_SHAPE issue, got %v", issues)
	}
}

func TestExtractComponentRefMarksZeroGameObjectAsPresent(t *testing.T) {
	body := "" +
		"Transform:\n" +
		"  m_GameObject: {fileID: 0}\n"

	component, _, issues := extractComponentRef(4000, body, 4, "Transform")

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if !component.HasGameObject {
		t.Fatalf("expected game object presence for inline fileID 0")
	}
	if component.GameObject != 0 {
		t.Fatalf("expected game object 0, got %d", component.GameObject)
	}
}

func TestExtractComponentRefUnknownGameObjectShapeReturnsIssue(t *testing.T) {
	body := string(loadGraphFixture(t, "unknown_gameobject_ref_shape.prefab"))

	component, _, issues := extractComponentRef(4000, body, 4, "Transform")

	if component.HasGameObject {
		t.Fatalf("expected missing game object ref")
	}
	if len(issues) != 1 || issues[0].Code != core.IssueUnknownFieldShape {
		t.Fatalf("expected UNKNOWN_FIELD_SHAPE issue, got %v", issues)
	}
}

func TestExtractComponentRefMarksExplicitZeroGameObjectAsPresent(t *testing.T) {
	body := "" +
		"Transform:\n" +
		"  m_GameObject: {fileID: 0}\n"

	component, _, issues := extractComponentRef(4000, body, 4, "Transform")

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if !component.HasGameObject {
		t.Fatalf("expected HasGameObject to be true for explicit {fileID: 0}")
	}
	if component.GameObject != 0 {
		t.Fatalf("expected explicit zero game object, got %d", component.GameObject)
	}
}

func loadGraphFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %q: %v", name, err)
	}

	return data
}
