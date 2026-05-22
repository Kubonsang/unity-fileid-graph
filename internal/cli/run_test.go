package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
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

func loadGolden(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "golden", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return string(data)
}
