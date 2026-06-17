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

// TestParseChildFileIDListAcceptsSameIndentDash covers gap1: Unity's real
// serialization puts the child dash at the SAME indent as m_Children (F3). The
// list must parse and stop exactly at the next sibling key (m_Father).
func TestParseChildFileIDListAcceptsSameIndentDash(t *testing.T) {
	body := "" +
		"  m_Children:\n" +
		"  - {fileID: 4001}\n" +
		"  - {fileID: 4002}\n" +
		"  m_Father: {fileID: 0}\n"

	lines, issues := scanBodyLines(body, 4000)
	if len(issues) != 0 {
		t.Fatalf("expected no scan issues, got %v", issues)
	}
	children, ok := parseChildFileIDList(lines, 0)
	if !ok {
		t.Fatalf("expected same-indent (F3) child list to parse")
	}
	if len(children) != 2 || children[0] != 4001 || children[1] != 4002 {
		t.Fatalf("unexpected children (must stop at m_Father): %v", children)
	}
}

// TestParseChildFileIDListIgnoresDeeperNestedDash is the directIndent regression
// guard (net-negative ①): a dash deeper than the first child's indent belongs to
// a multiline child item and must NOT be over-collected as a separate child.
func TestParseChildFileIDListIgnoresDeeperNestedDash(t *testing.T) {
	body := "" +
		"m_Children:\n" +
		"- {fileID: 100}\n" +
		"- {fileID: 200}\n" +
		"  - {fileID: 999}\n" + // deeper than directIndent(0): not a sibling child
		"m_Father: {fileID: 0}\n"

	lines, _ := scanBodyLines(body, 4000)
	children, ok := parseChildFileIDList(lines, 0)
	if !ok {
		t.Fatalf("expected child list to parse")
	}
	if len(children) != 2 || children[0] != 100 || children[1] != 200 {
		t.Fatalf("directIndent guard failed; over-collected deeper dash: %v", children)
	}
	for _, c := range children {
		if c == 999 {
			t.Fatalf("deeper nested dash 999 was wrongly collected: %v", children)
		}
	}
}

// TestParseChildFileIDListEmptyBlockYieldsNoChildren covers F6: an m_Children key
// with no entries (then a sibling key) is a valid empty list, not UNKNOWN.
func TestParseChildFileIDListEmptyBlockYieldsNoChildren(t *testing.T) {
	body := "" +
		"m_Children:\n" +
		"m_Father: {fileID: 0}\n"

	lines, _ := scanBodyLines(body, 4000)
	children, ok := parseChildFileIDList(lines, 0)
	if !ok {
		t.Fatalf("expected empty (F6) block to parse as empty children")
	}
	if len(children) != 0 {
		t.Fatalf("expected no children, got %v", children)
	}
}

// TestParseChildFileIDListRejectsUnknownDeeperContent keeps the fail-safe: deeper
// non-dash content under m_Children with no parseable child entry stays UNKNOWN.
func TestParseChildFileIDListRejectsUnknownDeeperContent(t *testing.T) {
	body := "" +
		"m_Children:\n" +
		"  garbage: x\n" +
		"m_Father: {fileID: 0}\n"

	lines, _ := scanBodyLines(body, 4000)
	if _, ok := parseChildFileIDList(lines, 0); ok {
		t.Fatalf("expected unknown deeper content to be rejected (UNKNOWN_FIELD_SHAPE)")
	}
}
