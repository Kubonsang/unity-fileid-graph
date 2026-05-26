package roundtrip

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

func TestRunLosslessCopyReturnsOKForHealthyFixture(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "fixtures", "check_ok.prefab")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "check_ok.copy.prefab")
	result, err := RunLosslessCopy(input, outputPath)
	if err != nil {
		t.Fatalf("RunLosslessCopy returned error: %v", err)
	}

	if result.Status != core.RoundtripStatusOK {
		t.Fatalf("expected %q, got %q", core.RoundtripStatusOK, result.Status)
	}
	if !result.BytesEqual || !result.Reparsed || !result.BlockSequenceEqual {
		t.Fatalf("expected all verification flags true, got %+v", result)
	}
	if result.GraphCheckStatus != core.CheckStatusOK {
		t.Fatalf("expected graph-check OK, got %q", result.GraphCheckStatus)
	}
	if result.EditorOpenStatus != core.EditorOpenNotChecked {
		t.Fatalf("expected editor open NOT_CHECKED, got %q", result.EditorOpenStatus)
	}
}

func TestRunLosslessCopyPreservesCRLFFixture(t *testing.T) {
	inputPath := filepath.Join("..", "..", "testdata", "fixtures", "roundtrip_crlf.prefab")
	input, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "roundtrip_crlf.copy.prefab")
	result, err := RunLosslessCopy(input, outputPath)
	if err != nil {
		t.Fatalf("RunLosslessCopy returned error: %v", err)
	}

	if result.LineEndingStyle != "CRLF" {
		t.Fatalf("expected CRLF, got %q", result.LineEndingStyle)
	}
	if !result.BytesEqual {
		t.Fatalf("expected byte-identical CRLF copy")
	}
}
