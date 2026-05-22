package core

import "testing"

func TestGraphPreservesDuplicateFileIDEvidence(t *testing.T) {
	graph := &Graph{
		Blocks: []*Block{
			{FileID: 1000, ClassID: 1},
			{FileID: 1000, ClassID: 114},
		},
		BlocksByID: map[int64][]*Block{
			1000: {
				{FileID: 1000, ClassID: 1},
				{FileID: 1000, ClassID: 114},
			},
		},
	}

	if len(graph.BlocksByID[1000]) != 2 {
		t.Fatalf("expected duplicate blocks to be preserved, got %d", len(graph.BlocksByID[1000]))
	}
}
