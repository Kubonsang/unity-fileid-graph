package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
	"github.com/Kubonsang/unity-fileid-graph/internal/parser"
)

func TestBuildPreservesDuplicateFileIDBlocks(t *testing.T) {
	parsed := parseGraphFixture(t, "duplicate_fileid_graph.prefab")

	graph, err := Build(parsed)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	blocks := graph.BlocksByID[1000]
	if len(blocks) != 2 {
		t.Fatalf("expected duplicate block evidence, got %d", len(blocks))
	}
	if blocks[0] == blocks[1] {
		t.Fatalf("expected distinct duplicate block pointers")
	}

	objects := graph.ObjectsByID[1000]
	if len(objects) != 2 {
		t.Fatalf("expected duplicate object evidence, got %d", len(objects))
	}
	if objects[0].TypeName != "GameObject" || objects[1].TypeName != "MonoBehaviour" {
		t.Fatalf("unexpected duplicate object types: %q %q", objects[0].TypeName, objects[1].TypeName)
	}
}

func TestBuildNilParseResultReturnsInitializedEmptyGraph(t *testing.T) {
	graph, err := Build(nil)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if graph == nil {
		t.Fatalf("expected initialized graph")
	}
	if len(graph.Blocks) != 0 || len(graph.Issues) != 0 {
		t.Fatalf("expected empty graph state, got %+v", graph)
	}
	if graph.BlocksByID == nil || graph.ObjectsByID == nil || graph.GameObjects == nil || graph.Components == nil || graph.Transforms == nil {
		t.Fatalf("expected initialized maps on empty graph, got %+v", graph)
	}
}

func TestBuildMaterialFixtureReportsUnknownClassID(t *testing.T) {
	parsed := parseGraphFixture(t, "material.mat")

	graph, err := Build(parsed)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if len(graph.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(graph.Blocks))
	}
	if len(graph.Issues) != 1 || graph.Issues[0].Code != core.IssueUnknownClassID {
		t.Fatalf("expected UNKNOWN_CLASS_ID issue, got %v", graph.Issues)
	}
	if got := graph.ObjectsByID[2100000]; len(got) != 1 {
		t.Fatalf("expected object evidence for unknown class, got %d", len(got))
	}
}

func TestBuildGraphPrefabBuildsReadOnlyGraphFromParserOutput(t *testing.T) {
	parsed := parseGraphFixture(t, "graph_prefab.prefab")

	graph, err := Build(parsed)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if len(graph.Issues) != 0 {
		t.Fatalf("expected no issues, got %v", graph.Issues)
	}

	goNode := graph.GameObjects[1000]
	if goNode == nil {
		t.Fatalf("expected game object 1000")
	}
	if goNode.Name != "Player" {
		t.Fatalf("expected name Player, got %q", goNode.Name)
	}
	if len(goNode.Components) != 2 {
		t.Fatalf("expected 2 component refs, got %v", goNode.Components)
	}
	if goNode.Components[0] != 11400000 || goNode.Components[1] != 4000 {
		t.Fatalf("expected source-order component refs, got %v", goNode.Components)
	}
	if goNode.Transform != 4000 {
		t.Fatalf("expected transform fileID 4000, got %d", goNode.Transform)
	}

	transformComponent := graph.Components[4000]
	if transformComponent == nil {
		t.Fatalf("expected transform component 4000")
	}
	if transformComponent.TypeName != "Transform" || !transformComponent.HasGameObject || transformComponent.GameObject != 1000 {
		t.Fatalf("unexpected transform component: %+v", transformComponent)
	}

	mono := graph.Components[11400000]
	if mono == nil {
		t.Fatalf("expected monobehaviour component 11400000")
	}
	if mono.TypeName != "MonoBehaviour" || !mono.HasGameObject || mono.GameObject != 1000 {
		t.Fatalf("unexpected monobehaviour component: %+v", mono)
	}
	if mono.Script == nil {
		t.Fatalf("expected script metadata")
	}
	if mono.Script.FileID != 11500000 || mono.Script.GUID != "0123456789abcdef0123456789abcdef" || mono.Script.Type != 3 {
		t.Fatalf("unexpected script metadata: %+v", mono.Script)
	}

	transform := graph.Transforms[4000]
	if transform == nil {
		t.Fatalf("expected transform node 4000")
	}
	if transform.GameObject != 1000 || transform.Father != 0 {
		t.Fatalf("unexpected transform linkages: %+v", transform)
	}
	if len(transform.Children) != 2 || transform.Children[0] != 4001 || transform.Children[1] != 4002 {
		t.Fatalf("unexpected transform children: %v", transform.Children)
	}
}

func TestBuildSimpleScenePreservesEmptyChildrenWithoutIssue(t *testing.T) {
	parsed := parseGraphFixture(t, "simple_scene.unity")

	graph, err := Build(parsed)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if len(graph.Issues) != 0 {
		t.Fatalf("expected no issues, got %v", graph.Issues)
	}

	transform := graph.Transforms[4001]
	if transform == nil {
		t.Fatalf("expected transform node 4001")
	}
	if len(transform.Children) != 0 {
		t.Fatalf("expected empty child list, got %v", transform.Children)
	}
}

func TestBuildMonoBehaviourFixtureKeepsExternalScriptMetadata(t *testing.T) {
	parsed := parseGraphFixture(t, "monobehaviour.prefab")

	graph, err := Build(parsed)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if len(graph.Issues) != 0 {
		t.Fatalf("expected no issues, got %v", graph.Issues)
	}

	component := graph.Components[11400000]
	if component == nil {
		t.Fatalf("expected monobehaviour component")
	}
	if component.Script == nil {
		t.Fatalf("expected external script metadata")
	}
	if component.Script.FileID != 11500000 || component.Script.GUID != "fedcba9876543210fedcba9876543210" || component.Script.Type != 3 {
		t.Fatalf("unexpected script metadata: %+v", component.Script)
	}
}

func TestBuildTabIndentFixtureKeepsPartialGraphAndIssue(t *testing.T) {
	parsed := parseGraphFixture(t, "tab_indent.prefab")

	graph, err := Build(parsed)
	if err != nil {
		t.Fatalf("unexpected build error: %v", err)
	}

	if len(graph.Issues) != 1 || graph.Issues[0].Code != core.IssueTabIndent {
		t.Fatalf("expected TAB_INDENT issue, got %v", graph.Issues)
	}
	goNode := graph.GameObjects[1000]
	if goNode == nil {
		t.Fatalf("expected partial game object node")
	}
	if goNode.Name != "TabIndented" {
		t.Fatalf("expected partial extraction to keep tab-indented name, got %q", goNode.Name)
	}
}

func parseGraphFixture(t *testing.T, name string) *core.ParseResult {
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

	return parsed
}
