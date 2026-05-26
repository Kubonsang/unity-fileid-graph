package core

import "testing"

func TestRoundtripResultRecomputeStatusOK(t *testing.T) {
	result := &RoundtripResult{
		Mode:               RoundtripModeLosslessBlockCopy,
		OutputPath:         "/tmp/out.prefab",
		BytesEqual:         true,
		Reparsed:           true,
		BlockSequenceEqual: true,
		GraphCheckStatus:   CheckStatusOK,
		LineEndingStyle:    "LF",
		EditorOpenStatus:   EditorOpenNotChecked,
	}

	result.RecomputeStatus()

	if result.Status != RoundtripStatusOK {
		t.Fatalf("expected %q, got %q", RoundtripStatusOK, result.Status)
	}
}

func TestRoundtripResultRecomputeStatusErrorWhenAnyVerificationFails(t *testing.T) {
	result := &RoundtripResult{
		Mode:               RoundtripModeLosslessBlockCopy,
		OutputPath:         "/tmp/out.prefab",
		BytesEqual:         false,
		Reparsed:           true,
		BlockSequenceEqual: true,
		GraphCheckStatus:   CheckStatusOK,
		LineEndingStyle:    "LF",
		EditorOpenStatus:   EditorOpenNotChecked,
	}

	result.RecomputeStatus()

	if result.Status != RoundtripStatusError {
		t.Fatalf("expected %q, got %q", RoundtripStatusError, result.Status)
	}
}
