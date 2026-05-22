package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"unity-fileid-graph/internal/core"
)

func TestParseHeaderParsesNormalHeader(t *testing.T) {
	header := "--- !u!114 &11400000\n"

	meta, err := parseHeader(header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.ClassID != 114 {
		t.Fatalf("expected class id 114, got %d", meta.ClassID)
	}
	if meta.FileID != 11400000 {
		t.Fatalf("expected file id 11400000, got %d", meta.FileID)
	}
	if meta.IsStripped {
		t.Fatalf("expected stripped to be false")
	}
}

func TestParseHeaderParsesStrippedHeader(t *testing.T) {
	header := "--- !u!1 &101000 stripped\r\n"

	meta, err := parseHeader(header)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.ClassID != 1 {
		t.Fatalf("expected class id 1, got %d", meta.ClassID)
	}
	if meta.FileID != 101000 {
		t.Fatalf("expected file id 101000, got %d", meta.FileID)
	}
	if !meta.IsStripped {
		t.Fatalf("expected stripped to be true")
	}
}

func TestParseHeaderRejectsInvalidHeader(t *testing.T) {
	header := "--- not-unity-header\n"

	if _, err := parseHeader(header); err == nil {
		t.Fatalf("expected error for invalid header")
	}
}

func TestParseHeaderRejectsSignedFileIDSyntax(t *testing.T) {
	header := "--- !u!114 &-123\n"

	if _, err := parseHeader(header); err == nil {
		t.Fatalf("expected error for signed file id syntax")
	}
}

func TestParseHeaderRejectsMissingTrailingNewline(t *testing.T) {
	header := "--- !u!114 &123"

	if _, err := parseHeader(header); err == nil {
		t.Fatalf("expected error for missing trailing newline")
	}
}

func TestParseBlocks(t *testing.T) {
	t.Run("preserves preamble bodies order and trailer across two blocks", func(t *testing.T) {
		input := []byte(
			"%YAML 1.1\n" +
				"%TAG !u! tag:unity3d.com,2011:\n" +
				"--- !u!1 &1000\n" +
				"GameObject:\n" +
				"  m_Name: First\n" +
				"--- !u!114 &2000 stripped\r\n" +
				"MonoBehaviour:\r\n" +
				"  m_Name: Second\r\n" +
				"...\r\n",
		)

		result, err := Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := &core.ParseResult{
			PreambleRaw: "%YAML 1.1\n%TAG !u! tag:unity3d.com,2011:\n",
			TrailerRaw:  "...\r\n",
			Blocks: []*core.Block{
				{
					Index:      0,
					ClassID:    1,
					FileID:     1000,
					HeaderRaw:  "--- !u!1 &1000\n",
					BodyRaw:    "GameObject:\n  m_Name: First\n",
					IsStripped: false,
				},
				{
					Index:      1,
					ClassID:    114,
					FileID:     2000,
					HeaderRaw:  "--- !u!114 &2000 stripped\r\n",
					BodyRaw:    "MonoBehaviour:\r\n  m_Name: Second\r\n",
					IsStripped: true,
				},
			},
		}

		assertParseResultEqual(t, result, expected)
	})

	t.Run("allows duplicate file ids", func(t *testing.T) {
		input := []byte(
			"--- !u!1 &1000\n" +
				"GameObject:\n" +
				"--- !u!114 &1000\n" +
				"MonoBehaviour:\n",
		)

		result, err := Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Blocks) != 2 {
			t.Fatalf("expected 2 blocks, got %d", len(result.Blocks))
		}
		if result.Blocks[0].FileID != 1000 || result.Blocks[1].FileID != 1000 {
			t.Fatalf("expected duplicate file ids to be preserved, got %d and %d", result.Blocks[0].FileID, result.Blocks[1].FileID)
		}
	})

	t.Run("keeps indented header like text inside body", func(t *testing.T) {
		input := []byte(
			"--- !u!1 &1000\n" +
				"MonoBehaviour:\n" +
				"  m_Script: |\n" +
				"  --- !u!999 &777\n" +
				"  tail\n",
		)

		result, err := Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := &core.ParseResult{
			Blocks: []*core.Block{
				{
					Index:      0,
					ClassID:    1,
					FileID:     1000,
					HeaderRaw:  "--- !u!1 &1000\n",
					BodyRaw:    "MonoBehaviour:\n  m_Script: |\n  --- !u!999 &777\n  tail\n",
					IsStripped: false,
				},
			},
		}

		assertParseResultEqual(t, result, expected)
	})

	t.Run("preserves LF fixture line endings", func(t *testing.T) {
		input := loadFixture(t, "lf_prefab.prefab")

		result, err := Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Blocks) != 1 {
			t.Fatalf("expected 1 block, got %d", len(result.Blocks))
		}
		if strings.Contains(result.Blocks[0].HeaderRaw, "\r") || strings.Contains(result.Blocks[0].BodyRaw, "\r") {
			t.Fatalf("expected LF fixture to preserve LF-only content")
		}
		if result.TrailerRaw != "" {
			t.Fatalf("expected no trailer for LF fixture, got %q", result.TrailerRaw)
		}
	})

	t.Run("preserves CRLF fixture line endings", func(t *testing.T) {
		input := loadFixture(t, "crlf_prefab.prefab")

		result, err := Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Blocks) != 1 {
			t.Fatalf("expected 1 block, got %d", len(result.Blocks))
		}
		if !strings.Contains(result.Blocks[0].HeaderRaw, "\r\n") {
			t.Fatalf("expected CRLF header to be preserved, got %q", result.Blocks[0].HeaderRaw)
		}
		if !strings.Contains(result.Blocks[0].BodyRaw, "\r\n") {
			t.Fatalf("expected CRLF body to be preserved, got %q", result.Blocks[0].BodyRaw)
		}
		if result.TrailerRaw != "" {
			t.Fatalf("expected no trailer for CRLF fixture, got %q", result.TrailerRaw)
		}
	})

	t.Run("moves document end marker into trailer", func(t *testing.T) {
		input := loadFixture(t, "document_end_marker.prefab")

		result, err := Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Blocks) != 1 {
			t.Fatalf("expected 1 block, got %d", len(result.Blocks))
		}
		if strings.Contains(result.Blocks[0].BodyRaw, "...") {
			t.Fatalf("expected document marker removed from body, got %q", result.Blocks[0].BodyRaw)
		}
		if result.TrailerRaw != "...\n" {
			t.Fatalf("expected trailer %q, got %q", "...\n", result.TrailerRaw)
		}
	})

	t.Run("rejects invalid header fixture", func(t *testing.T) {
		input := loadFixture(t, "invalid_header.prefab")

		_, err := Parse(input)
		if err == nil {
			t.Fatalf("expected parse error")
		}
		if !strings.Contains(err.Error(), "invalid Unity header") {
			t.Fatalf("expected invalid header error, got %v", err)
		}
	})

	t.Run("rejects malformed attempted unity header without valid blocks", func(t *testing.T) {
		input := []byte(
			"--- !u!1 1000\n" +
				"...\n",
		)

		_, err := Parse(input)
		if err == nil {
			t.Fatalf("expected parse error")
		}
		if !strings.Contains(err.Error(), "invalid Unity header") {
			t.Fatalf("expected invalid header error, got %v", err)
		}
	})

	t.Run("keeps unrelated top-level document markers in preamble", func(t *testing.T) {
		input := []byte(
			"--- note\n" +
				"plain text\n" +
				"--- !u!1 &1000\n" +
				"GameObject:\n",
		)

		result, err := Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.PreambleRaw != "--- note\nplain text\n" {
			t.Fatalf("expected unrelated preamble to be preserved, got %q", result.PreambleRaw)
		}
		if len(result.Blocks) != 1 {
			t.Fatalf("expected 1 block, got %d", len(result.Blocks))
		}
	})

	t.Run("keeps duplicate file ids from fixture", func(t *testing.T) {
		input := loadFixture(t, "duplicate_fileid.prefab")

		result, err := Parse(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(result.Blocks) != 2 {
			t.Fatalf("expected 2 blocks, got %d", len(result.Blocks))
		}
		if result.Blocks[0].FileID != result.Blocks[1].FileID {
			t.Fatalf("expected duplicate file ids, got %d and %d", result.Blocks[0].FileID, result.Blocks[1].FileID)
		}
	})
}

func assertParseResultEqual(t *testing.T, got *core.ParseResult, want *core.ParseResult) {
	t.Helper()

	if got == nil {
		t.Fatalf("expected parse result, got nil")
	}
	if got.PreambleRaw != want.PreambleRaw {
		t.Fatalf("expected preamble %q, got %q", want.PreambleRaw, got.PreambleRaw)
	}
	if got.TrailerRaw != want.TrailerRaw {
		t.Fatalf("expected trailer %q, got %q", want.TrailerRaw, got.TrailerRaw)
	}
	if len(got.Blocks) != len(want.Blocks) {
		t.Fatalf("expected %d blocks, got %d", len(want.Blocks), len(got.Blocks))
	}

	for i := range want.Blocks {
		gotBlock := got.Blocks[i]
		wantBlock := want.Blocks[i]
		if gotBlock.Index != wantBlock.Index {
			t.Fatalf("block %d: expected index %d, got %d", i, wantBlock.Index, gotBlock.Index)
		}
		if gotBlock.ClassID != wantBlock.ClassID {
			t.Fatalf("block %d: expected class id %d, got %d", i, wantBlock.ClassID, gotBlock.ClassID)
		}
		if gotBlock.FileID != wantBlock.FileID {
			t.Fatalf("block %d: expected file id %d, got %d", i, wantBlock.FileID, gotBlock.FileID)
		}
		if gotBlock.HeaderRaw != wantBlock.HeaderRaw {
			t.Fatalf("block %d: expected header %q, got %q", i, wantBlock.HeaderRaw, gotBlock.HeaderRaw)
		}
		if gotBlock.BodyRaw != wantBlock.BodyRaw {
			t.Fatalf("block %d: expected body %q, got %q", i, wantBlock.BodyRaw, gotBlock.BodyRaw)
		}
		if gotBlock.IsStripped != wantBlock.IsStripped {
			t.Fatalf("block %d: expected stripped %v, got %v", i, wantBlock.IsStripped, gotBlock.IsStripped)
		}
	}
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "fixtures", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return data
}
