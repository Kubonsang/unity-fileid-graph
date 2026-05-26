package mutate

import (
	"errors"
	"strings"
	"testing"
)

func TestExtractComponentOwnerGameObjectReadsInlinePPtr(t *testing.T) {
	body := "BoxCollider:\n  m_GameObject: {fileID: 1000}\n  m_Enabled: 1\n"

	goID, err := ExtractComponentOwnerGameObject(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if goID != 1000 {
		t.Fatalf("expected owner 1000, got %d", goID)
	}
}

func TestRemoveComponentEntryDeletesExactlyOneSingleLineListItem(t *testing.T) {
	body := "GameObject:\n  m_Component:\n  - component: {fileID: 4000}\n  - component: {fileID: 65000}\n"

	edited, err := RemoveComponentEntry(body, 65000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(edited, "65000") {
		t.Fatalf("expected target entry to be removed, got %q", edited)
	}
	if !strings.Contains(edited, "4000") {
		t.Fatalf("expected unrelated entry to remain")
	}
}

func TestRemoveComponentEntryRejectsMultilineComponentShape(t *testing.T) {
	body := "GameObject:\n  m_Component:\n  - component:\n      fileID: 65000\n"

	_, err := RemoveComponentEntry(body, 65000)
	if !errors.Is(err, ErrUnsupportedFieldShape) {
		t.Fatalf("expected unsupported field shape, got %v", err)
	}
}

func TestRemoveComponentEntryReturnsComponentRefNotFound(t *testing.T) {
	body := "GameObject:\n  m_Component:\n  - component: {fileID: 4000}\n"

	_, err := RemoveComponentEntry(body, 65000)
	if !errors.Is(err, ErrComponentRefNotFound) {
		t.Fatalf("expected component ref not found, got %v", err)
	}
}

func TestHasExactComponentEntryFindsTarget(t *testing.T) {
	body := "GameObject:\n  m_Component:\n  - component: {fileID: 4000}\n  - component: {fileID: 65000}\n"

	ok, err := HasExactComponentEntry(body, 65000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected exact component entry")
	}
}

func TestHasExactComponentEntryRejectsUnsupportedShape(t *testing.T) {
	body := "GameObject:\n  m_Component:\n  - component:\n      fileID: 65000\n"

	_, err := HasExactComponentEntry(body, 65000)
	if !errors.Is(err, ErrUnsupportedFieldShape) {
		t.Fatalf("expected unsupported field shape, got %v", err)
	}
}
