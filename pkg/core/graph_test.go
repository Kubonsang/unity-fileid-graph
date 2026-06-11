package core

import "testing"

func blocksByID(blocks []*Block) map[int64][]*Block {
	byID := make(map[int64][]*Block, len(blocks))
	for _, block := range blocks {
		byID[block.FileID] = append(byID[block.FileID], block)
	}
	return byID
}

func objectsByID(objects []*UnityObject) map[int64][]*UnityObject {
	byID := make(map[int64][]*UnityObject, len(objects))
	for _, object := range objects {
		byID[object.FileID] = append(byID[object.FileID], object)
	}
	return byID
}

func TestGraphPreservesDuplicateFileIDEvidence(t *testing.T) {
	graph := &Graph{
		Blocks: []*Block{
			{FileID: 1000, ClassID: 1},
			{FileID: 1000, ClassID: 114},
		},
	}
	graph.BlocksByID = blocksByID(graph.Blocks)

	duplicates := graph.BlocksByID[1000]
	if len(duplicates) != 2 {
		t.Fatalf("expected duplicate blocks to be preserved, got %d", len(duplicates))
	}
	if duplicates[0] == duplicates[1] {
		t.Fatalf("expected duplicate blocks to remain distinguishable pointers")
	}
	if duplicates[0].ClassID != 1 || duplicates[1].ClassID != 114 {
		t.Fatalf("expected duplicate evidence to preserve distinct class ids, got %d and %d", duplicates[0].ClassID, duplicates[1].ClassID)
	}
}

func TestGraphPreservesDuplicateObjectEvidence(t *testing.T) {
	graph := &Graph{
		ObjectsByID: map[int64][]*UnityObject{},
	}

	objects := []*UnityObject{
		{FileID: 1000, ClassID: 1, TypeName: "GameObject"},
		{FileID: 1000, ClassID: 114, TypeName: "MonoBehaviour"},
	}
	graph.ObjectsByID = objectsByID(objects)

	duplicates := graph.ObjectsByID[1000]
	if len(duplicates) != 2 {
		t.Fatalf("expected duplicate objects to be preserved, got %d", len(duplicates))
	}
	if duplicates[0] == duplicates[1] {
		t.Fatalf("expected duplicate objects to remain distinguishable pointers")
	}
	if duplicates[0].TypeName != "GameObject" || duplicates[1].TypeName != "MonoBehaviour" {
		t.Fatalf("expected duplicate object evidence to preserve distinct types, got %q and %q", duplicates[0].TypeName, duplicates[1].TypeName)
	}
}
