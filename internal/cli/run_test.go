package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

func TestRunRejectsMissingArguments(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{}, stdout, stderr)

	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if got := stderr.String(); got == "" {
		t.Fatalf("expected usage text on stderr")
	}
}

func TestRunRejectsUnknownNamespace(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"unknown", "blocks", "test.prefab"}, stdout, stderr)

	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if got := stderr.String(); got == "" {
		t.Fatalf("expected usage text on stderr")
	}
}

func TestRunMatchesGoldenSimplePrefab(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "blocks", "../../testdata/fixtures/simple_prefab.prefab"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("expected empty stderr, got %q", got)
	}

	wantStdout := loadGolden(t, "simple_prefab.blocks.txt")
	if got := stdout.String(); got != wantStdout {
		t.Fatalf("unexpected stdout:\nwant %q\ngot  %q", wantStdout, got)
	}
}

func TestRunMatchesGoldenStrippedHeader(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "blocks", "../../testdata/fixtures/stripped_header.prefab"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	if got := stderr.String(); got != "" {
		t.Fatalf("expected empty stderr, got %q", got)
	}

	wantStdout := loadGolden(t, "stripped_header.blocks.txt")
	if got := stdout.String(); got != wantStdout {
		t.Fatalf("unexpected stdout:\nwant %q\ngot  %q", wantStdout, got)
	}
}

func TestRunMatchesGoldenNegativeHeader(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "blocks", "../../testdata/fixtures/negative_header.prefab"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if got := stdout.String(); got != loadGolden(t, "negative_header.blocks.txt") {
		t.Fatalf("blocks golden mismatch:\nwant %q\ngot  %q", loadGolden(t, "negative_header.blocks.txt"), got)
	}
}

func TestRunReturnsReadErrorForUnreadableInput(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "blocks", t.TempDir()}, stdout, stderr)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("expected empty stdout, got %q", got)
	}

	if got := stderr.String(); got == "" || !bytes.Contains([]byte(got), []byte("read ")) {
		t.Fatalf("expected clear read error on stderr, got %q", got)
	}
}

func TestRunReturnsParseErrorForInvalidHeaderFixture(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "blocks", "../../testdata/fixtures/invalid_header.prefab"}, stdout, stderr)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}

	if got := stdout.String(); got != "" {
		t.Fatalf("expected empty stdout, got %q", got)
	}

	if got := stderr.String(); got == "" || !bytes.Contains([]byte(got), []byte("parse ")) {
		t.Fatalf("expected clear parse error on stderr, got %q", got)
	}
}

func TestRunMatchesGoldenGraphPrefab(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "graph", "../../testdata/fixtures/graph_prefab.prefab"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if got := stdout.String(); got != loadGolden(t, "graph_prefab.graph.txt") {
		t.Fatalf("graph golden mismatch:\nwant %q\ngot  %q", loadGolden(t, "graph_prefab.graph.txt"), got)
	}
}

func TestRunGraphPrintsWarningsAndKeepsExitCodeZero(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "graph", "../../testdata/fixtures/tab_indent.prefab"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0 for warning-only graph, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if got := stdout.String(); got != loadGolden(t, "tab_indent.graph.txt") {
		t.Fatalf("warning golden mismatch:\nwant %q\ngot  %q", loadGolden(t, "tab_indent.graph.txt"), got)
	}
}

func TestRunMatchesGoldenCheckOK(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "check", "../../testdata/fixtures/check_ok.prefab"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if got := stdout.String(); got != loadGolden(t, "check_ok.check.txt") {
		t.Fatalf("check golden mismatch:\nwant %q\ngot  %q", loadGolden(t, "check_ok.check.txt"), got)
	}
}

func TestRunCheckJSONMatchesGolden(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "check", "../../testdata/fixtures/check_ok.prefab", "--json"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	want := loadGolden(t, "check_ok.check.json")
	if stdout.String() != want {
		t.Fatalf("unexpected stdout:\nwant %q\ngot  %q", want, stdout.String())
	}
}

func TestRunMatchesGoldenDuplicateFileIDCheck(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "check", "../../testdata/fixtures/check_duplicate_fileid.prefab"}, stdout, stderr)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d: %s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if got := stdout.String(); got != loadGolden(t, "check_duplicate_fileid.check.txt") {
		t.Fatalf("check golden mismatch:\nwant %q\ngot  %q", loadGolden(t, "check_duplicate_fileid.check.txt"), got)
	}
}

func TestRunMatchesGoldenWarnOnlyCheck(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "check", "../../testdata/fixtures/tab_indent.prefab"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	if got := stdout.String(); got != loadGolden(t, "check_tab_indent.check.txt") {
		t.Fatalf("check golden mismatch:\nwant %q\ngot  %q", loadGolden(t, "check_tab_indent.check.txt"), got)
	}
}

func TestRunRefsMatchesGoldenPrefab(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "refs", "../../testdata/fixtures/refs_prefab.prefab"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	want := loadGolden(t, "refs_prefab.refs.txt")
	if stdout.String() != want {
		t.Fatalf("unexpected stdout:\nwant %q\ngot  %q", want, stdout.String())
	}
}

func TestRunRefsWarnExitsZeroAndMatchesGolden(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "refs", "../../testdata/fixtures/refs_warn_overflow.prefab"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0 for refs WARN, got %d stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	want := loadGolden(t, "refs_warn_overflow.refs.txt")
	if stdout.String() != want {
		t.Fatalf("unexpected stdout:\nwant %q\ngot  %q", want, stdout.String())
	}
}

func TestRunRefsJSONMatchesGolden(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "refs", "../../testdata/fixtures/refs_prefab.prefab", "--json"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	want := loadGolden(t, "refs_prefab.refs.json")
	if stdout.String() != want {
		t.Fatalf("unexpected stdout:\nwant %q\ngot  %q", want, stdout.String())
	}
}

func TestRunRefsJSONWarnExitsZeroAndMatchesGolden(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "refs", "../../testdata/fixtures/refs_warn_overflow.prefab", "--json"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0 for refs WARN json, got %d stderr=%q", exitCode, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
	want := loadGolden(t, "refs_warn_overflow.refs.json")
	if stdout.String() != want {
		t.Fatalf("unexpected stdout:\nwant %q\ngot  %q", want, stdout.String())
	}
}

func TestRunBlocksRejectsJSONInV09a(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "blocks", "../../testdata/fixtures/simple_prefab.prefab", "--json"}, stdout, stderr)

	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage output, got %q", stderr.String())
	}
}

func TestRunGraphRejectsJSONInV09a(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "graph", "../../testdata/fixtures/graph_prefab.prefab", "--json"}, stdout, stderr)

	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d stdout=%q stderr=%q", exitCode, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage output, got %q", stderr.String())
	}
}

func TestRunRoundtripWritesOutputAndPrintsSummary(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	outPath := filepath.Join(t.TempDir(), "check_ok.copy.prefab")

	exitCode := Run([]string{"prefab", "roundtrip", "../../testdata/fixtures/check_ok.prefab", "--out", outPath}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", exitCode, stderr.String())
	}
	want := "ROUNDTRIP status=OK mode=lossless-block-copy bytes_equal=1 reparsed=1 block_sequence_equal=1 graph_check=OK line_endings=LF editor_open=NOT_CHECKED out=" + outPath + "\n"
	if stdout.String() != want {
		t.Fatalf("unexpected stdout:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunRoundtripRejectsUnsupportedMode(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	outPath := filepath.Join(t.TempDir(), "check_ok.copy.prefab")

	exitCode := Run([]string{"prefab", "roundtrip", "../../testdata/fixtures/check_ok.prefab", "--out", outPath, "--mode", "yaml-node-serialize"}, stdout, stderr)

	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage output, got %q", stderr.String())
	}
}

func TestRunRoundtripReturnsWarnStatusWithZeroExitCode(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	outPath := filepath.Join(t.TempDir(), "material.copy.mat")

	exitCode := Run([]string{"mat", "roundtrip", "../../testdata/fixtures/material.mat", "--out", outPath}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "ROUNDTRIP status=WARN") {
		t.Fatalf("expected WARN roundtrip output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "graph_check=WARN") {
		t.Fatalf("expected graph_check=WARN, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunSetMutatesFixtureCopy(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_prefab.prefab")
	target := filepath.Join(t.TempDir(), "set_prefab.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	exitCode := Run([]string{"prefab", "set", target, "--id", "1000", "--field", "m_IsActive", "--value", "0"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "SET status=OK file_id=1000 field=m_IsActive old=1 new=0") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunSetReturnsBlockedExitZero(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_monobehaviour_blocked.prefab")
	target := filepath.Join(t.TempDir(), "set_monobehaviour_blocked.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	exitCode := Run([]string{"prefab", "set", target, "--id", "11400000", "--field", "m_Enabled", "--value", "0"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "SET status=BLOCKED code=MONOBEHAVIOUR_NATIVE_WRITE_BLOCKED") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestWriteSetPrintsMessageOnErrorStatus(t *testing.T) {
	stdout := &bytes.Buffer{}
	result := &core.SetResult{
		Status:     core.MutationStatusError,
		Code:       core.MutationCodeFinalCheckError,
		FileID:     1000,
		Field:      "m_IsActive",
		OldValue:   "1",
		NewValue:   "0",
		PreCheck:   core.CheckStatusOK,
		TempCheck:  core.CheckStatusOK,
		FinalCheck: core.CheckStatusError,
		BackupPath: "/tmp/sample.prefab.bak",
		Message:    "restored=true",
	}

	exitCode := writeSet(stdout, result)

	if exitCode != 1 {
		t.Fatalf("expected exit 1, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), `message="restored=true"`) {
		t.Fatalf("expected restore message in stdout, got %q", stdout.String())
	}
}

func TestRunSetRejectsMissingValueFlagWithoutMutating(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	source := filepath.Join("..", "..", "testdata", "fixtures", "set_prefab.prefab")
	target := filepath.Join(t.TempDir(), "set_prefab.prefab")
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	exitCode := Run([]string{"prefab", "set", target, "--id", "1000", "--field", "m_IsActive", "--field", "m_Name"}, stdout, stderr)

	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d stderr=%q stdout=%q", exitCode, stderr.String(), stdout.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage output, got %q", stderr.String())
	}
	after, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(after) != string(input) {
		t.Fatalf("expected file to remain unchanged on malformed set command")
	}
}

func TestRunRemoveComponentRejectsMissingExperimentalFlag(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "remove-component", "../../testdata/fixtures/remove_component_ok.prefab", "--id", "65000", "--write"}, stdout, stderr)

	if exitCode != 2 {
		t.Fatalf("expected usage exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("expected usage text, got %q", stderr.String())
	}
}

func TestRunRemoveComponentRejectsSceneNamespace(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"scene", "remove-component", "../../testdata/fixtures/remove_component_ok.prefab", "--id", "65000", "--experimental", "--write"}, stdout, stderr)

	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "uyaml prefab remove-component") {
		t.Fatalf("expected prefab-only usage guidance, got %q", stderr.String())
	}
}

func TestRunRemoveComponentPrintsExperimentalSuccessLine(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	target := copyCLIFixture(t, "remove_component_ok.prefab")

	exitCode := Run([]string{"prefab", "remove-component", target, "--id", "65000", "--experimental", "--write"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "REMOVE_COMPONENT status=EXPERIMENTAL file_id=65000 class_id=65 type=BoxCollider game_object=1000") {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestRunRemoveComponentPrintsBlockedMonoBehaviour(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	exitCode := Run([]string{"prefab", "remove-component", "../../testdata/fixtures/remove_component_monobehaviour_blocked.prefab", "--id", "11400000", "--experimental", "--write"}, stdout, stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit 0 for blocked result, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "status=BLOCKED") {
		t.Fatalf("expected blocked output, got %q", stdout.String())
	}
}

func TestWriteRemoveComponentPrintsMessageOnErrorStatus(t *testing.T) {
	stdout := &bytes.Buffer{}
	exitCode := writeRemoveComponent(stdout, &core.RemoveComponentResult{
		Status:     core.MutationStatusError,
		Code:       core.MutationCodeFinalCheckError,
		FileID:     65000,
		ClassID:    65,
		TypeName:   "BoxCollider",
		GameObject: 1000,
		PreCheck:   core.CheckStatusOK,
		TempCheck:  core.CheckStatusOK,
		FinalCheck: core.CheckStatusError,
		BackupPath: "/tmp/remove_component_ok.prefab.bak",
		Message:    "restore_failed=true",
	})

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "code=FINAL_CHECK_ERROR") {
		t.Fatalf("expected code field in output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), `message="restore_failed=true"`) {
		t.Fatalf("expected message field in output, got %q", stdout.String())
	}
}

func TestWriteRoundtripReturnsErrorExitForFailedVerification(t *testing.T) {
	stdout := &bytes.Buffer{}
	exitCode := writeRoundtrip(stdout, &core.RoundtripResult{
		Status:             core.RoundtripStatusError,
		Mode:               core.RoundtripModeLosslessBlockCopy,
		OutputPath:         "/tmp/out.prefab",
		BytesEqual:         false,
		Reparsed:           true,
		BlockSequenceEqual: true,
		GraphCheckStatus:   core.CheckStatusOK,
		LineEndingStyle:    "LF",
		EditorOpenStatus:   core.EditorOpenNotChecked,
	})

	if exitCode != 1 {
		t.Fatalf("expected exit 1, got %d", exitCode)
	}
}

func TestWriteGraphPrintsUnknownTypeForUnsupportedComponentRef(t *testing.T) {
	graphResult := &core.Graph{
		GameObjects: map[int64]*core.GameObjectNode{
			1000: {FileID: 1000, Name: "Player", Components: []int64{23000}},
		},
		Components: map[int64]*core.ComponentNode{},
		Transforms: map[int64]*core.TransformNode{},
		Issues: []core.Issue{
			{
				Code:    core.IssueUnsupportedComponentRef,
				FileID:  23000,
				Message: "component referenced by GameObject but not extracted in v0.2",
			},
		},
	}

	stdout := &bytes.Buffer{}
	exitCode := writeGraph(stdout, graphResult)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	output := stdout.String()
	if !strings.Contains(output, "component=23000 type=UNKNOWN") {
		t.Fatalf("expected UNKNOWN component output, got %q", output)
	}
	if !strings.Contains(output, "WARN code=UNSUPPORTED_COMPONENT_REF file_id=23000") {
		t.Fatalf("expected warning output, got %q", output)
	}
}

func TestWriteGraphPrintsUnknownGameObjectWhenRefMissing(t *testing.T) {
	graphResult := &core.Graph{
		GameObjects: map[int64]*core.GameObjectNode{},
		Components: map[int64]*core.ComponentNode{
			11400000: {
				FileID:        11400000,
				ClassID:       114,
				TypeName:      "MonoBehaviour",
				HasGameObject: false,
			},
		},
		Transforms: map[int64]*core.TransformNode{},
	}

	stdout := &bytes.Buffer{}
	exitCode := writeGraph(stdout, graphResult)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "COMPONENT id=11400000 type=MonoBehaviour game_object=UNKNOWN") {
		t.Fatalf("expected UNKNOWN game object output, got %q", stdout.String())
	}
}

func TestWriteGraphPrintsExplicitZeroGameObjectWhenPresent(t *testing.T) {
	graphResult := &core.Graph{
		GameObjects: map[int64]*core.GameObjectNode{},
		Components: map[int64]*core.ComponentNode{
			4000: {
				FileID:        4000,
				ClassID:       4,
				TypeName:      "Transform",
				HasGameObject: true,
				GameObject:    0,
			},
		},
		Transforms: map[int64]*core.TransformNode{},
	}

	stdout := &bytes.Buffer{}
	exitCode := writeGraph(stdout, graphResult)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if !strings.Contains(stdout.String(), "COMPONENT id=4000 type=Transform game_object=0") {
		t.Fatalf("expected explicit zero game object output, got %q", stdout.String())
	}
}

func TestWriteGraphOrdersSectionsDeterministicallyAndPreservesComponentOrder(t *testing.T) {
	graphResult := &core.Graph{
		GameObjects: map[int64]*core.GameObjectNode{
			2000: {FileID: 2000, Name: "Second", Components: []int64{33000, 11000}},
			1000: {FileID: 1000, Name: "First", Components: []int64{22000}},
		},
		Components: map[int64]*core.ComponentNode{
			11000: {FileID: 11000, TypeName: "Transform", HasGameObject: true, GameObject: 2000},
			22000: {FileID: 22000, TypeName: "MeshRenderer", HasGameObject: true, GameObject: 1000},
			33000: {FileID: 33000, TypeName: "MonoBehaviour", HasGameObject: true, GameObject: 2000},
		},
		Transforms: map[int64]*core.TransformNode{
			9000: {FileID: 9000, GameObject: 2000},
			4000: {FileID: 4000, GameObject: 1000},
		},
	}

	stdout := &bytes.Buffer{}
	exitCode := writeGraph(stdout, graphResult)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	want := "" +
		"GAMEOBJECT id=1000 name=First\n" +
		"  component=22000 type=MeshRenderer\n" +
		"\n" +
		"GAMEOBJECT id=2000 name=Second\n" +
		"  component=33000 type=MonoBehaviour\n" +
		"  component=11000 type=Transform\n" +
		"\n" +
		"COMPONENT id=11000 type=Transform game_object=2000\n" +
		"\n" +
		"COMPONENT id=22000 type=MeshRenderer game_object=1000\n" +
		"\n" +
		"COMPONENT id=33000 type=MonoBehaviour game_object=2000\n" +
		"\n" +
		"TRANSFORM id=4000 game_object=1000 father=0 children=none\n" +
		"TRANSFORM id=9000 game_object=2000 father=0 children=none\n"
	if got := stdout.String(); got != want {
		t.Fatalf("unexpected deterministic graph output:\nwant %q\ngot  %q", want, got)
	}
}

func loadGolden(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "golden", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(data)
}

func copyCLIFixture(t *testing.T, name string) string {
	t.Helper()

	source := filepath.Join("..", "..", "testdata", "fixtures", name)
	target := filepath.Join(t.TempDir(), name)
	input, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	if err := os.WriteFile(target, input, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	return target
}
