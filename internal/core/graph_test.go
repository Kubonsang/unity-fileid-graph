package core

import "testing"

func blocksByID(blocks []*Block) map[int64][]*Block {
	byID := make(map[int64][]*Block, len(blocks))
	for _, block := range blocks {
		byID[block.FileID] = append(byID[block.FileID], block)
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
