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
	validateGameObjectComponentBackrefs(graphResult, result)
	validateTransformParentChildRelationships(graphResult, result)
	validateMissingTransformComponents(graphResult, result)
	validateSuspiciousMonoBehaviourScripts(graphResult, result)
	appendGraphWarnings(graphResult, result)

	result.RecomputeStatus()
	return result
}

func validateDuplicateFileIDs(graphResult *core.Graph, result *core.CheckResult) {
	for _, fileID := range sortedBlockIDs(graphResult.BlocksByID) {
		if len(graphResult.BlocksByID[fileID]) < 2 {
			continue
		}
		result.Errors = append(result.Errors, core.CheckFinding{
			Code:           core.CheckDuplicateFileID,
			DuplicateCount: len(graphResult.BlocksByID[fileID]),
			FileID:         fileID,
			Reason:         "duplicate_block_headers",
		})
	}
}

func validateMissingComponentBlocks(graphResult *core.Graph, result *core.CheckResult) {
	for _, gameObjectID := range sortedGameObjectIDs(graphResult.GameObjects) {
		if hasGraphIssueForFile(graphResult, gameObjectID) {
			continue
		}
		gameObject := graphResult.GameObjects[gameObjectID]
		for _, componentID := range gameObject.Components {
			if component, ok := graphResult.Components[componentID]; ok && component != nil {
				continue
			}
			if hasObjectBlock(graphResult, componentID) {
				if hasComponentObjectBlock(graphResult, componentID) {
					continue
				}
				result.Errors = append(result.Errors, core.CheckFinding{
					Code:         core.CheckMissingComponentBlock,
					GameObjectID: gameObjectID,
					ComponentID:  componentID,
					Reason:       "referenced_block_is_not_component",
				})
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

func validateGameObjectComponentBackrefs(graphResult *core.Graph, result *core.CheckResult) {
	for _, gameObjectID := range sortedGameObjectIDs(graphResult.GameObjects) {
		if gameObjectHasRelatedIssues(graphResult, gameObjectID) {
			continue
		}
		gameObject := graphResult.GameObjects[gameObjectID]
		for _, componentID := range gameObject.Components {
			if hasGraphIssueForFile(graphResult, componentID) {
				continue
			}
			component, ok := graphResult.Components[componentID]
			if !ok || component == nil {
				continue
			}

			reason := ""
			switch {
			case !component.HasGameObject || component.GameObject == 0:
				reason = "component_missing_gameobject"
			case component.GameObject != gameObjectID:
				reason = "component_points_to_other_gameobject"
			}

			if reason == "" {
				continue
			}
			result.Errors = append(result.Errors, core.CheckFinding{
				Code:         core.CheckGoComponentBackrefMismatch,
				GameObjectID: gameObjectID,
				ComponentID:  componentID,
				Reason:       reason,
			})
		}
	}
}

func validateTransformParentChildRelationships(graphResult *core.Graph, result *core.CheckResult) {
	for _, transformID := range sortedTransformIDs(graphResult.Transforms) {
		if hasGraphIssueForFile(graphResult, transformID) {
			continue
		}
		transform := graphResult.Transforms[transformID]
		if transform == nil {
			continue
		}

		if transform.Father != 0 {
			parentTransform, ok := graphResult.Transforms[transform.Father]
			if !ok || parentTransform == nil {
				result.Errors = append(result.Errors, core.CheckFinding{
					Code:        core.CheckTransformParentChildMismatch,
					TransformID: transformID,
					ParentID:    transform.Father,
					Reason:      "missing_parent_transform",
				})
			} else if !hasGraphIssueForFile(graphResult, transform.Father) && !containsInt64(parentTransform.Children, transformID) {
				result.Errors = append(result.Errors, core.CheckFinding{
					Code:        core.CheckTransformParentChildMismatch,
					TransformID: transformID,
					ParentID:    transform.Father,
					Reason:      "missing_from_parent_children",
				})
			}
		}

		for _, childID := range transform.Children {
			if hasGraphIssueForFile(graphResult, childID) {
				continue
			}
			childTransform, ok := graphResult.Transforms[childID]
			if !ok || childTransform == nil {
				result.Errors = append(result.Errors, core.CheckFinding{
					Code:        core.CheckTransformParentChildMismatch,
					TransformID: transformID,
					ChildID:     childID,
					Reason:      "missing_child_transform",
				})
				continue
			}
			if childTransform.Father != transformID {
				result.Errors = append(result.Errors, core.CheckFinding{
					Code:        core.CheckTransformParentChildMismatch,
					TransformID: transformID,
					ParentID:    childTransform.Father,
					ChildID:     childID,
					Reason:      "child_father_mismatch",
				})
			}
		}
	}
}

func validateMissingTransformComponents(graphResult *core.Graph, result *core.CheckResult) {
	for _, gameObjectID := range sortedGameObjectIDs(graphResult.GameObjects) {
		if gameObjectHasRelatedIssues(graphResult, gameObjectID) {
			continue
		}
		gameObject := graphResult.GameObjects[gameObjectID]
		if gameObject == nil || gameObject.Transform != 0 {
			continue
		}
		result.Errors = append(result.Errors, core.CheckFinding{
			Code:         core.CheckMissingTransformComponent,
			GameObjectID: gameObjectID,
			Reason:       "missing_transform_component",
		})
	}
}

func validateSuspiciousMonoBehaviourScripts(graphResult *core.Graph, result *core.CheckResult) {
	for _, componentID := range sortedComponentIDs(graphResult.Components) {
		component := graphResult.Components[componentID]
		if component == nil || component.TypeName != "MonoBehaviour" {
			continue
		}
		if !isSuspiciousScript(component.Script) {
			continue
		}
		result.Errors = append(result.Errors, core.CheckFinding{
			Code:        core.CheckSuspiciousMonoBehaviourScript,
			ComponentID: componentID,
			Reason:      "missing_script_metadata",
		})
	}
}

func appendGraphWarnings(graphResult *core.Graph, result *core.CheckResult) {
	for _, issue := range graphResult.Issues {
		result.Warnings = append(result.Warnings, core.CheckFinding{
			Code:    issue.Code,
			FileID:  issue.FileID,
			Message: issue.Message,
		})
	}
}

func isSuspiciousScript(script *core.ScriptRef) bool {
	if script == nil {
		return true
	}
	if script.FileID == 0 || script.GUID == "" {
		return true
	}
	return script.Type <= 0
}

func hasGraphIssueForFile(graphResult *core.Graph, fileID int64) bool {
	for _, issue := range graphResult.Issues {
		if issue.FileID == fileID {
			return true
		}
	}
	return false
}

func hasObjectBlock(graphResult *core.Graph, fileID int64) bool {
	return len(graphResult.ObjectsByID[fileID]) > 0
}

func hasComponentObjectBlock(graphResult *core.Graph, fileID int64) bool {
	for _, object := range graphResult.ObjectsByID[fileID] {
		if object == nil {
			continue
		}
		if isKnownComponentClassID(object.ClassID) {
			return true
		}
	}
	return false
}

func isKnownComponentClassID(classID int) bool {
	switch classID {
	case 4, 23, 33, 54, 65, 114:
		return true
	default:
		return false
	}
}

func containsInt64(values []int64, want int64) bool {
	return slices.Contains(values, want)
}

func gameObjectHasRelatedIssues(graphResult *core.Graph, gameObjectID int64) bool {
	if hasGraphIssueForFile(graphResult, gameObjectID) {
		return true
	}

	gameObject := graphResult.GameObjects[gameObjectID]
	if gameObject == nil {
		return false
	}

	for _, componentID := range gameObject.Components {
		if hasGraphIssueForFile(graphResult, componentID) {
			return true
		}
	}

	if gameObject.Transform != 0 && hasGraphIssueForFile(graphResult, gameObject.Transform) {
		return true
	}

	return false
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

func sortedTransformIDs(transforms map[int64]*core.TransformNode) []int64 {
	keys := make([]int64, 0, len(transforms))
	for fileID := range transforms {
		keys = append(keys, fileID)
	}
	slices.Sort(keys)
	return keys
}
