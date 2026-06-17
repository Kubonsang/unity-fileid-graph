package mutate

import (
	"errors"
	"os"

	"github.com/Kubonsang/unity-fileid-graph/internal/roundtrip"
	"github.com/Kubonsang/unity-fileid-graph/pkg/check"
	"github.com/Kubonsang/unity-fileid-graph/pkg/core"
	"github.com/Kubonsang/unity-fileid-graph/pkg/graph"
	"github.com/Kubonsang/unity-fileid-graph/pkg/parser"
)

var removableComponentTypes = map[int]string{
	54: "Rigidbody",
	65: "BoxCollider",
}

func RunRemoveComponent(opts core.RemoveComponentOptions) (*core.RemoveComponentResult, error) {
	return runRemoveComponentWithDeps(opts, defaultFileOps(), writePipelineOptions{
		RestoreOnFinalCheckError: true,
	})
}

func runRemoveComponentWithDeps(opts core.RemoveComponentOptions, ops fileOps, pipelineOptions writePipelineOptions) (*core.RemoveComponentResult, error) {
	input, err := os.ReadFile(opts.InputPath)
	if err != nil {
		return nil, err
	}
	parsed, err := parser.Parse(input)
	if err != nil {
		return nil, err
	}
	graphResult, err := graph.Build(parsed)
	if err != nil {
		return nil, err
	}
	pre := check.Run(graphResult)

	result := &core.RemoveComponentResult{
		FileID:                        opts.FileID,
		PreCheck:                      pre.Status,
		PreCheckSkippedLinks:          pre.SkippedLinks,
		PreCheckSkippedStripped:       pre.SkippedStripped,
		PreCheckSkippedUnmodeledClass: pre.SkippedUnmodeledClass,
	}
	if !opts.Experimental {
		result.Code = core.MutationCodeExperimentalFlagRequired
		result.MarkBlocked("remove-component requires --experimental")
		return result, nil
	}
	if !opts.Write {
		result.Code = core.MutationCodeWriteFlagRequired
		result.MarkBlocked("remove-component requires --write")
		return result, nil
	}

	targetBlock, code := FindUniqueBlockByFileID(parsed, opts.FileID)
	if code != "" {
		result.Code = code
		result.MarkBlocked("mutation target is missing or ambiguous")
		return result, nil
	}
	result.ClassID = targetBlock.ClassID

	if targetBlock.IsStripped {
		result.Code = core.MutationCodeStrippedObjectBlocked
		result.MarkBlocked("native structural writes to stripped objects are blocked in v0.6")
		return result, nil
	}
	if targetBlock.ClassID == 4 {
		result.Code = core.MutationCodeTransformRemoveBlocked
		result.MarkBlocked("removing Transform is blocked in v0.6")
		return result, nil
	}
	if targetBlock.ClassID == 114 {
		result.Code = core.MutationCodeMonoBehaviourWriteBlocked
		result.MarkBlocked("native remove-component on MonoBehaviour is blocked in v0.6")
		return result, nil
	}

	typeName, ok := removableComponentTypes[targetBlock.ClassID]
	if !ok {
		typeName = "UNKNOWN"
		message := "native remove-component is limited to the v0.6 built-in allowlist"
		switch targetBlock.ClassID {
		case 23:
			typeName = "MeshRenderer"
			message = "MeshRenderer removal stays blocked because sibling MeshFilter dependency handling is not implemented"
		case 33:
			typeName = "MeshFilter"
			message = "MeshFilter removal stays blocked because sibling MeshRenderer dependency handling is not implemented"
		}
		result.Code = core.MutationCodeUnsupportedComponentClass
		result.TypeName = typeName
		result.MarkBlocked(message)
		return result, nil
	}
	result.TypeName = typeName

	goID, err := ExtractComponentOwnerGameObject(targetBlock.BodyRaw)
	if err != nil {
		result.Code = core.MutationCodeComponentOwnerNotFound
		result.MarkBlocked(err.Error())
		return result, nil
	}
	result.GameObject = goID

	goBlock, goCode := FindUniqueBlockByFileID(parsed, goID)
	if goCode != "" || goBlock == nil || goBlock.ClassID != 1 {
		result.Code = core.MutationCodeComponentOwnerNotFound
		result.MarkBlocked("owner GameObject block is missing or ambiguous")
		return result, nil
	}

	hasEntry, err := HasExactComponentEntry(goBlock.BodyRaw, opts.FileID)
	if err != nil {
		result.Code = core.MutationCodeUnsupportedComponentList
		result.MarkBlocked(err.Error())
		return result, nil
	}
	if !hasEntry {
		result.Code = core.MutationCodeComponentOwnerMismatch
		result.MarkBlocked("target component owner and GameObject component list do not match")
		return result, nil
	}
	if pre.Status == core.CheckStatusError {
		result.Code = core.MutationCodePrecheckError
		result.MarkBlocked("graph-check before mutation returned ERROR")
		return result, nil
	}

	editedBody, err := RemoveComponentEntry(goBlock.BodyRaw, opts.FileID)
	if err != nil {
		result.Code = core.MutationCodeComponentOwnerMismatch
		if errors.Is(err, ErrUnsupportedFieldShape) {
			result.Code = core.MutationCodeUnsupportedComponentList
		} else if errors.Is(err, ErrComponentRefNotFound) {
			result.Code = core.MutationCodeComponentOwnerMismatch
		}
		result.MarkBlocked(err.Error())
		return result, nil
	}
	goBlock.BodyRaw = editedBody

	editedParsed, err := DropBlockByFileID(parsed, opts.FileID)
	if err != nil {
		return nil, err
	}
	if remainingBlocksContainFileIDReference(editedParsed, opts.FileID) {
		result.Code = core.MutationCodeDanglingFileID
		result.MarkBlocked("remove-component would leave dangling local fileID references")
		return result, nil
	}

	output := roundtrip.AssembleLosslessCopy(editedParsed)
	pipeline, err := completeWritePipeline(opts.InputPath, output, ops, pipelineOptions)
	if err != nil {
		return nil, err
	}
	result.TempCheck = pipeline.TempCheck
	result.FinalCheck = pipeline.FinalCheck
	result.BackupPath = pipeline.BackupPath
	if result.TempCheck == core.CheckStatusError {
		result.Code = core.MutationCodeTempCheckError
		result.RecomputeStatus()
		return result, nil
	}
	if result.FinalCheck == core.CheckStatusError {
		result.Code = core.MutationCodeFinalCheckError
		if pipeline.Restored {
			result.Message = "restored=true"
		}
		if pipeline.RestoreFail {
			result.Message = "restore_failed=true"
		}
	}

	result.RecomputeStatus()
	return result, nil
}
