package graph

import (
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/pkg/core"
)

func TestScanBodyLinesKeepsEffectiveText(t *testing.T) {
	body := "" +
		"m_Children:\n" +
		"  - {fileID: 4001}\n"

	lines, issues := scanBodyLines(body, 4000)

	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[1].EffectiveText != "{fileID: 4001}" {
		t.Fatalf("expected effective text %q, got %q", "{fileID: 4001}", lines[1].EffectiveText)
	}
}

func TestParseChildFileIDList(t *testing.T) {
	body := "" +
		"m_Children:\n" +
		"  - {fileID: 4001}\n" +
		"  - {fileID: 4002}\n"

	lines, issues := scanBodyLines(body, 4000)
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %v", issues)
	}

	children, ok := parseChildFileIDList(lines, 0)
	if !ok {
		t.Fatalf("expected child list parse to succeed")
	}
	if len(children) != 2 || children[0] != 4001 || children[1] != 4002 {
		t.Fatalf("unexpected children: %v", children)
	}
}

func TestParseNestedFileIDStopsAtSiblingBoundary(t *testing.T) {
	lines := []bodyLine{
		{Indent: 0, EffectiveKey: "m_GameObject"},
		{Indent: 2, EffectiveKey: "guid", RawValue: "abc"},
		{Indent: 0, EffectiveKey: "m_Name", RawValue: "Player"},
	}

	if _, ok := parseNestedFileID(lines, 0); ok {
		t.Fatalf("expected nested parse to stop at sibling boundary")
	}
}

func TestScanBodyLinesReportsTabIndent(t *testing.T) {
	body := "GameObject:\n\tm_Name: TabIndented\n"

	_, issues := scanBodyLines(body, 1000)

	if len(issues) != 1 || issues[0].Code != core.IssueTabIndent {
		t.Fatalf("expected TAB_INDENT issue, got %v", issues)
	}
}
