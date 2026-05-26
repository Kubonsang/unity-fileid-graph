package roundtrip

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/internal/parser"
)

func TestAssembleLosslessCopyMatchesOriginalFixtureBytes(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "fixtures", "simple_prefab.prefab")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	parsed, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	output := AssembleLosslessCopy(parsed)
	if string(output) != string(input) {
		t.Fatalf("expected byte-identical copy")
	}
}

func TestAssembleLosslessCopyPreservesPreambleAndTrailer(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "fixtures", "roundtrip_preamble.prefab")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	parsed, err := parser.Parse(input)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	output := AssembleLosslessCopy(parsed)
	if string(output) != string(input) {
		t.Fatalf("expected byte-identical copy with preamble and trailer")
	}
}

func TestDetectLineEndingReturnsCRLF(t *testing.T) {
	if got := DetectLineEnding([]byte("--- !u!1 &1000\r\nGameObject:\r\n")); got != "CRLF" {
		t.Fatalf("expected CRLF, got %q", got)
	}
}
