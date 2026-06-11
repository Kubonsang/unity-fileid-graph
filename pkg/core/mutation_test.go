package core

import "testing"

func TestSetResultRecomputeStatusReturnsWarnForWarnOnlyMutation(t *testing.T) {
	result := &SetResult{
		FileID:     2100000,
		Field:      "m_Name",
		OldValue:   "Body",
		NewValue:   "\"Helmet\"",
		PreCheck:   CheckStatusWarn,
		TempCheck:  CheckStatusWarn,
		FinalCheck: CheckStatusWarn,
		BackupPath: "/tmp/set_material.mat.bak",
	}

	result.RecomputeStatus()

	if result.Status != MutationStatusWarn {
		t.Fatalf("expected %q, got %q", MutationStatusWarn, result.Status)
	}
}

func TestSetResultRecomputeStatusReturnsBlockedWhenCodePresent(t *testing.T) {
	result := &SetResult{
		FileID: 11400000,
		Field:  "m_Enabled",
		Code:   MutationCodeMonoBehaviourWriteBlocked,
	}

	result.MarkBlocked("native scalar writes to MonoBehaviour are blocked in v0.5")

	if result.Status != MutationStatusBlocked {
		t.Fatalf("expected %q, got %q", MutationStatusBlocked, result.Status)
	}
}

func TestRemoveComponentResultRecomputeStatusKeepsExperimentalForWarnOnlyChecks(t *testing.T) {
	result := &RemoveComponentResult{
		FileID:     65000,
		ClassID:    65,
		TypeName:   "BoxCollider",
		GameObject: 1000,
		PreCheck:   CheckStatusWarn,
		TempCheck:  CheckStatusWarn,
		FinalCheck: CheckStatusWarn,
	}

	result.RecomputeStatus()

	if result.Status != MutationStatusExperimental {
		t.Fatalf("expected %q, got %q", MutationStatusExperimental, result.Status)
	}
}

func TestRemoveComponentResultBlockedStatusWins(t *testing.T) {
	result := &RemoveComponentResult{
		FileID:   4000,
		ClassID:  4,
		TypeName: "Transform",
		Code:     MutationCodeTransformRemoveBlocked,
		Status:   MutationStatusBlocked,
	}

	result.RecomputeStatus()

	if result.Status != MutationStatusBlocked {
		t.Fatalf("expected blocked status, got %q", result.Status)
	}
}
