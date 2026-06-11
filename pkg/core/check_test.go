package core

import "testing"

func TestCheckResultSupportsErrorAndWarnStates(t *testing.T) {
	result := &CheckResult{
		Status:          CheckStatusWarn,
		BlockCount:      1,
		GameObjectCount: 1,
		ComponentCount:  0,
		TransformCount:  0,
		Errors:          []CheckFinding{},
		Warnings:        []CheckFinding{{Code: CheckSuspiciousMonoBehaviourScript, ComponentID: 11400000, Reason: "missing_script"}},
	}

	if result.Status != CheckStatusWarn {
		t.Fatalf("expected status %q, got %q", CheckStatusWarn, result.Status)
	}

	if len(result.Errors) != 0 {
		t.Fatalf("expected no errors, got %v", result.Errors)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(result.Warnings))
	}
	if result.Warnings[0].Code != CheckSuspiciousMonoBehaviourScript {
		t.Fatalf("expected warning code %q, got %q", CheckSuspiciousMonoBehaviourScript, result.Warnings[0].Code)
	}
}

func TestCheckResultRecomputeStatusPrefersErrorsThenWarnings(t *testing.T) {
	result := &CheckResult{
		Errors:   []CheckFinding{},
		Warnings: []CheckFinding{},
	}

	result.RecomputeStatus()
	if result.Status != CheckStatusOK {
		t.Fatalf("expected OK status, got %q", result.Status)
	}

	result.Warnings = append(result.Warnings, CheckFinding{Code: CheckSuspiciousMonoBehaviourScript})
	result.RecomputeStatus()
	if result.Status != CheckStatusWarn {
		t.Fatalf("expected WARN status, got %q", result.Status)
	}

	result.Errors = append(result.Errors, CheckFinding{Code: CheckDuplicateFileID})
	result.RecomputeStatus()
	if result.Status != CheckStatusError {
		t.Fatalf("expected ERROR status, got %q", result.Status)
	}
}

func TestCheckResultUsesSlicePlacementForSeverity(t *testing.T) {
	result := &CheckResult{
		Errors: []CheckFinding{
			{Code: CheckDuplicateFileID, FileID: 1000},
		},
		Warnings: []CheckFinding{
			{Code: CheckSuspiciousMonoBehaviourScript, ComponentID: 11400000},
		},
	}

	result.RecomputeStatus()
	if result.Status != CheckStatusError {
		t.Fatalf("expected ERROR status when errors exist, got %q", result.Status)
	}
	if result.Errors[0].Code != CheckDuplicateFileID {
		t.Fatalf("expected error code %q, got %q", CheckDuplicateFileID, result.Errors[0].Code)
	}
	if result.Warnings[0].Code != CheckSuspiciousMonoBehaviourScript {
		t.Fatalf("expected warning code %q, got %q", CheckSuspiciousMonoBehaviourScript, result.Warnings[0].Code)
	}
}

func TestRefsResultRecomputeStatusReturnsWarnForIssues(t *testing.T) {
	result := &RefsResult{
		Issues: []Issue{{Code: IssueUnknownFieldShape}},
	}

	result.RecomputeStatus()

	if result.Status != RefsStatusWarn {
		t.Fatalf("expected WARN, got %q", result.Status)
	}
}
