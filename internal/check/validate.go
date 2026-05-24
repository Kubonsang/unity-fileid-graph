package check

import (
	"slices"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

func Run(graphResult *core.Graph) *core.CheckResult {
	if graphResult == nil {
		result := &core.CheckResult{
			Errors:   []core.CheckFinding{},
			Warnings: []core.CheckFinding{},
		}
		result.RecomputeStatus()
		return result
	}

	result := &core.CheckResult{
		BlockCount:      len(graphResult.Blocks),
		GameObjectCount: len(graphResult.GameObjects),
		ComponentCount:  len(graphResult.Components),
		TransformCount:  len(graphResult.Transforms),
		Errors:          []core.CheckFinding{},
		Warnings:        []core.CheckFinding{},
	}

	validateDuplicateFileIDs(graphResult, result)
	validateMissingComponentBlocks(graphResult, result)
	validateMissingGameObjectBlocks(graphResult, result)

	result.RecomputeStatus()
	return result
}

func validateDuplicateFileIDs(graphResult *core.Graph, result *core.CheckResult) {
	for _, fileID := range sortedBlockIDs(graphResult.BlocksByID) {
		if len(graphResult.BlocksByID[fileID]) < 2 {
			continue
		}
		result.Errors = append(result.Errors, core.CheckFinding{
			Code:   core.CheckDuplicateFileID,
			FileID: fileID,
			Reason: "duplicate_block_headers",
		})
	}
}

func validateMissingComponentBlocks(graphResult *core.Graph, result *core.CheckResult) {
	for _, gameObjectID := range sortedGameObjectIDs(graphResult.GameObjects) {
		gameObject := graphResult.GameObjects[gameObjectID]
		for _, componentID := range gameObject.Components {
			if component, ok := graphResult.Components[componentID]; ok && component != nil {
				continue
			}
			result.Errors = append(result.Errors, core.CheckFinding{
				Code:         core.CheckMissingComponentBlock,
				GameObjectID: gameObjectID,
				ComponentID:  componentID,
				Reason:       "missing_component_block",
			})
		}
	}
}

func validateMissingGameObjectBlocks(graphResult *core.Graph, result *core.CheckResult) {
	for _, componentID := range sortedComponentIDs(graphResult.Components) {
		component := graphResult.Components[componentID]
		if component == nil || !component.HasGameObject || component.GameObject == 0 {
			continue
		}
		if _, ok := graphResult.GameObjects[component.GameObject]; ok {
			continue
		}
		result.Errors = append(result.Errors, core.CheckFinding{
			Code:         core.CheckMissingGameObjectBlock,
			GameObjectID: component.GameObject,
			ComponentID:  componentID,
			Reason:       "missing_gameobject_block",
		})
	}
}

func sortedBlockIDs(blocksByID map[int64][]*core.Block) []int64 {
	keys := make([]int64, 0, len(blocksByID))
	for fileID := range blocksByID {
		keys = append(keys, fileID)
	}
	slices.Sort(keys)
	return keys
}

func sortedGameObjectIDs(gameObjects map[int64]*core.GameObjectNode) []int64 {
	keys := make([]int64, 0, len(gameObjects))
	for fileID := range gameObjects {
		keys = append(keys, fileID)
	}
	slices.Sort(keys)
	return keys
}

func sortedComponentIDs(components map[int64]*core.ComponentNode) []int64 {
	keys := make([]int64, 0, len(components))
	for fileID := range components {
		keys = append(keys, fileID)
	}
	slices.Sort(keys)
	return keys
}
