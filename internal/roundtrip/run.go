package roundtrip

import (
	"bytes"
	"os"

	"github.com/Kubonsang/unity-fileid-graph/internal/check"
	"github.com/Kubonsang/unity-fileid-graph/internal/graph"
	"github.com/Kubonsang/unity-fileid-graph/pkg/core"
	"github.com/Kubonsang/unity-fileid-graph/pkg/parser"
)

func RunLosslessCopy(input []byte, outputPath string) (*core.RoundtripResult, error) {
	parsed, err := parser.Parse(input)
	if err != nil {
		return nil, err
	}

	output := AssembleLosslessCopy(parsed)
	if err := os.WriteFile(outputPath, output, 0o644); err != nil {
		return nil, err
	}

	reparsed, err := parser.Parse(output)
	if err != nil {
		return nil, err
	}

	graphResult, err := graph.Build(reparsed)
	if err != nil {
		return nil, err
	}
	checkResult := check.Run(graphResult)

	result := &core.RoundtripResult{
		Mode:               core.RoundtripModeLosslessBlockCopy,
		OutputPath:         outputPath,
		BytesEqual:         bytes.Equal(input, output),
		Reparsed:           true,
		BlockSequenceEqual: equalBlockSequence(parsed, reparsed),
		GraphCheckStatus:   checkResult.Status,
		LineEndingStyle:    DetectLineEnding(input),
		EditorOpenStatus:   core.EditorOpenNotChecked,
	}
	result.RecomputeStatus()
	return result, nil
}

func equalBlockSequence(left, right *core.ParseResult) bool {
	if len(left.Blocks) != len(right.Blocks) {
		return false
	}

	for i := range left.Blocks {
		lb := left.Blocks[i]
		rb := right.Blocks[i]
		if lb.Index != rb.Index {
			return false
		}
		if lb.ClassID != rb.ClassID {
			return false
		}
		if lb.FileID != rb.FileID {
			return false
		}
		if lb.IsStripped != rb.IsStripped {
			return false
		}
		if lb.HeaderRaw != rb.HeaderRaw {
			return false
		}
	}

	return true
}
