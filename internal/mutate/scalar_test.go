package mutate

import (
	"errors"
	"strings"
	"testing"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

func TestPlanScalarEditFindsTopLevelBoolField(t *testing.T) {
	body := "GameObject:\n  m_IsActive: 1\n  m_Name: Player\n"

	plan, err := PlanScalarEdit(body, "m_IsActive", "0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.OldValue != "1" {
		t.Fatalf("expected old value 1, got %q", plan.OldValue)
	}
	if plan.NewValue != "0" {
		t.Fatalf("expected new value 0, got %q", plan.NewValue)
	}
	if plan.NewBody == body {
		t.Fatalf("expected body to change")
	}
}

func TestPlanScalarEditRejectsInlineObjectField(t *testing.T) {
	body := "Transform:\n  m_GameObject: {fileID: 1000}\n"

	_, err := PlanScalarEdit(body, "m_GameObject", "0")
	if !errors.Is(err, ErrUnsupportedFieldShape) {
		t.Fatalf("expected unsupported field shape error, got %v", err)
	}
}

func TestPlanScalarEditFormatsStringAsQuotedYAML(t *testing.T) {
	body := "GameObject:\n  m_Name: Player\n"

	plan, err := PlanScalarEdit(body, "m_Name", "Boss One")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.NewValue != "\"Boss One\"" {
		t.Fatalf("expected quoted yaml string, got %q", plan.NewValue)
	}
}

func TestPlanScalarEditTreatsKnownStringFieldWithNumericOldValueAsString(t *testing.T) {
	body := "GameObject:\n  m_Name: 123\n"

	plan, err := PlanScalarEdit(body, "m_Name", "Boss")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.NewValue != "\"Boss\"" {
		t.Fatalf("expected quoted string replacement, got %q", plan.NewValue)
	}
}

func TestPlanScalarEditReturnsFieldNotFoundSeparately(t *testing.T) {
	body := "GameObject:\n  m_Name: Player\n"

	_, err := PlanScalarEdit(body, "m_Missing", "0")
	if !errors.Is(err, ErrFieldNotFound) {
		t.Fatalf("expected field not found, got %v", err)
	}
}

func TestPlanScalarEditTreatsRootOrderZeroAsIntNotBool(t *testing.T) {
	body := "Transform:\n  m_RootOrder: 0\n"

	plan, err := PlanScalarEdit(body, "m_RootOrder", "2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.NewValue != "2" {
		t.Fatalf("expected int replacement, got %q", plan.NewValue)
	}
}

func TestPlanScalarEditPreservesCRLFLineEndings(t *testing.T) {
	body := "GameObject:\r\n  m_IsActive: 1\r\n"

	plan, err := PlanScalarEdit(body, "m_IsActive", "0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(plan.NewBody, "\r\n") {
		t.Fatalf("expected CRLF to be preserved, got %q", plan.NewBody)
	}
}

func TestValidateBlockMutationRejectsMonoBehaviour(t *testing.T) {
	block := &core.Block{FileID: 11400000, ClassID: 114, BodyRaw: "MonoBehaviour:\n  m_Enabled: 1\n"}

	code, msg := ValidateBlockMutation(block)

	if code != core.MutationCodeMonoBehaviourWriteBlocked {
		t.Fatalf("expected MonoBehaviour blocked code, got %q", code)
	}
	if msg == "" {
		t.Fatalf("expected blocked message")
	}
}

func TestValidateBlockMutationRejectsStrippedObject(t *testing.T) {
	block := &core.Block{FileID: 1000, ClassID: 1, IsStripped: true}

	code, _ := ValidateBlockMutation(block)

	if code != core.MutationCodeStrippedObjectBlocked {
		t.Fatalf("expected stripped blocked code, got %q", code)
	}
}

func TestFindUniqueBlockByFileIDRejectsDuplicates(t *testing.T) {
	parsed := &core.ParseResult{
		Blocks: []*core.Block{
			{FileID: 1000, ClassID: 1},
			{FileID: 1000, ClassID: 4},
		},
	}

	block, code := FindUniqueBlockByFileID(parsed, 1000)
	if block != nil {
		t.Fatalf("expected nil block for duplicate file id")
	}
	if code != core.MutationCodeDuplicateFileID {
		t.Fatalf("expected duplicate code, got %q", code)
	}
}
