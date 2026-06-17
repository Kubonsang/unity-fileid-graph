package check

import (
	"slices"
	"strconv"
	"strings"

	"github.com/Kubonsang/unity-fileid-graph/pkg/core"
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
	validateTransformCycles(graphResult, result)
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
				if referencesKnownNonComponentBlock(graphResult, componentID) {
					result.Errors = append(result.Errors, core.CheckFinding{
						Code:         core.CheckMissingComponentBlock,
						GameObjectID: gameObjectID,
						ComponentID:  componentID,
						Reason:       "referenced_block_is_not_component",
					})
				}
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

// validateTransformParentChildRelationships asserts parent/child symmetry, but
// only between endpoints whose links are locally authoritative. A parent/child
// reference is SKIPPED (counted, never silently dropped) when the other endpoint
// is a stripped nested prefab-instance block or an unmodeled-class block (e.g.
// RectTransform 224) — for those the local file does not carry the data needed
// to assert a mismatch. A reference to a fileID with NO block at all is a genuine
// dangling link and is still reported as an ERROR.
func validateTransformParentChildRelationships(graphResult *core.Graph, result *core.CheckResult) {
	skipStripped := func() { result.SkippedLinks++; result.SkippedStripped++ }
	skipUnmodeled := func() { result.SkippedLinks++; result.SkippedUnmodeledClass++ }

	for _, transformID := range sortedTransformIDs(graphResult.Transforms) {
		if hasGraphIssueForFile(graphResult, transformID) {
			continue
		}
		// Stripped transforms are nested prefab-instance children: their
		// m_Father/m_Children are not stored locally (they live in the source
		// prefab + m_Modifications), so symmetry is not assertable for them.
		if isStrippedFileID(graphResult, transformID) {
			continue
		}
		transform := graphResult.Transforms[transformID]
		if transform == nil {
			continue
		}

		if transform.Father != 0 {
			parentTransform, ok := graphResult.Transforms[transform.Father]
			switch {
			case !ok || parentTransform == nil:
				// Father is not a modeled transform.
				switch {
				case isStrippedFileID(graphResult, transform.Father):
					skipStripped()
				case isUnmodeledTransformClass(graphResult, transform.Father):
					skipUnmodeled() // transform-like class not yet modeled (RectTransform 224)
				default:
					// No block, or a block whose class is not a transform at all:
					// a genuine dangling/wrong-type parent reference.
					result.Errors = append(result.Errors, core.CheckFinding{
						Code:        core.CheckTransformParentChildMismatch,
						TransformID: transformID,
						ParentID:    transform.Father,
						Reason:      "missing_parent_transform",
					})
				}
			case hasGraphIssueForFile(graphResult, transform.Father):
				// Pre-existing graph-issue skip (handled elsewhere).
			case isStrippedFileID(graphResult, transform.Father):
				skipStripped()
			case !containsInt64(parentTransform.Children, transformID):
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
			// A stripped child carries no local m_Father to compare against.
			if isStrippedFileID(graphResult, childID) {
				skipStripped()
				continue
			}
			childTransform, ok := graphResult.Transforms[childID]
			if !ok || childTransform == nil {
				// A transform-like-but-unmodeled child (RectTransform) is skipped;
				// a child fileID with no block, or a block that is not a transform
				// class at all, is a genuine dangling/wrong-type reference.
				if isUnmodeledTransformClass(graphResult, childID) {
					skipUnmodeled()
				} else {
					result.Errors = append(result.Errors, core.CheckFinding{
						Code:        core.CheckTransformParentChildMismatch,
						TransformID: transformID,
						ChildID:     childID,
						Reason:      "missing_child_transform",
					})
				}
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

// validateTransformCycles detects transform parent cycles. It runs directed-cycle
// detection independently on the father graph (t -> t.Father) and the children
// graph (t -> each child). A valid hierarchy is a tree: acyclic in BOTH
// directions, so a normal symmetric parent/child pair (A.Father=B and B lists A)
// is NOT a cycle. Only a real loop is — a father chain (A->B->C->A), a
// children chain, or a 2-node mutual link (A.Father=B and B.Father=A). A diamond
// (a node reachable two ways but with no back-edge) is not flagged.
//
// It reads Transforms[id].Father/.Children directly and does NOT short-circuit on
// hasGraphIssueForFile: a cycle must surface even when a node carries another
// issue (otherwise gap1-style issues could hide the very loop they create).
// Stripped / unmodeled-class endpoints simply contribute no edges (they are not
// in Transforms), so they never form or mask a cycle.
func validateTransformCycles(graphResult *core.Graph, result *core.CheckResult) {
	transformExists := func(id int64) bool {
		_, ok := graphResult.Transforms[id]
		return ok
	}
	fatherAdj := func(id int64) []int64 {
		t := graphResult.Transforms[id]
		if t == nil || t.Father == 0 || !transformExists(t.Father) {
			return nil
		}
		return []int64{t.Father}
	}
	childAdj := func(id int64) []int64 {
		t := graphResult.Transforms[id]
		if t == nil {
			return nil
		}
		out := make([]int64, 0, len(t.Children))
		for _, c := range t.Children {
			if transformExists(c) {
				out = append(out, c)
			}
		}
		return out
	}

	reported := map[string]bool{}
	for _, adj := range []func(int64) []int64{fatherAdj, childAdj} {
		for _, cycle := range findDirectedCycles(graphResult.Transforms, adj) {
			key := canonicalCycleKey(cycle)
			if reported[key] {
				continue
			}
			reported[key] = true
			result.Errors = append(result.Errors, core.CheckFinding{
				Code:        core.CheckTransformParentCycle,
				TransformID: minInt64(cycle),
				Reason:      "transform_parent_cycle",
				Message:     formatCycleChain(cycle),
			})
		}
	}
}

// findDirectedCycles returns the elementary cycles reachable in the directed
// graph defined by adj, deterministically (transform IDs visited in sorted
// order). Each returned slice lists the cycle's nodes in traversal order.
func findDirectedCycles(transforms map[int64]*core.TransformNode, adj func(int64) []int64) [][]int64 {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	color := make(map[int64]int, len(transforms))
	var stack []int64
	var cycles [][]int64

	var dfs func(int64)
	dfs = func(u int64) {
		color[u] = gray
		stack = append(stack, u)
		for _, v := range adj(u) {
			switch color[v] {
			case gray:
				// Back-edge: extract the cycle v..u from the stack.
				for i := len(stack) - 1; i >= 0; i-- {
					if stack[i] == v {
						cycles = append(cycles, append([]int64(nil), stack[i:]...))
						break
					}
				}
			case white:
				dfs(v)
			}
		}
		stack = stack[:len(stack)-1]
		color[u] = black
	}

	for _, id := range sortedTransformIDs(transforms) {
		if color[id] == white {
			dfs(id)
		}
	}
	return cycles
}

// canonicalCycleKey returns an order-independent identity for a cycle's node set,
// so the same loop found via the father and children graphs is reported once.
func canonicalCycleKey(cycle []int64) string {
	ids := append([]int64(nil), cycle...)
	slices.Sort(ids)
	var b strings.Builder
	for i, id := range ids {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatInt(id, 10))
	}
	return b.String()
}

func formatCycleChain(cycle []int64) string {
	var b strings.Builder
	b.WriteString("cycle=")
	for i, id := range cycle {
		if i > 0 {
			b.WriteString("->")
		}
		b.WriteString(strconv.FormatInt(id, 10))
	}
	if len(cycle) > 0 {
		b.WriteString("->")
		b.WriteString(strconv.FormatInt(cycle[0], 10))
	}
	return b.String()
}

func minInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v < m {
			m = v
		}
	}
	return m
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

// isStrippedFileID reports whether any block at fileID is a stripped (nested
// prefab-instance) block. Such blocks do not store local parent/child links.
func isStrippedFileID(graphResult *core.Graph, fileID int64) bool {
	for _, block := range graphResult.BlocksByID[fileID] {
		if block != nil && block.IsStripped {
			return true
		}
	}
	return false
}

// isUnmodeledTransformClass reports whether a block at fileID is a transform-like
// class that the graph does not yet model as a TransformNode. Today that is only
// RectTransform (224); Transform (4) is modeled. Such an endpoint cannot be
// symmetry-checked yet, so its link is skipped (and counted) rather than flagged.
// A non-transform-class block (GameObject, Material, …) is NOT covered here, so a
// parent/child reference to one stays a genuine mismatch error.
func isUnmodeledTransformClass(graphResult *core.Graph, fileID int64) bool {
	for _, block := range graphResult.BlocksByID[fileID] {
		if block != nil && block.ClassID == 224 {
			return true
		}
	}
	return false
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

// referencesKnownNonComponentBlock reports whether every block present at fileID
// is a class that can never be a GameObject component (e.g. GameObject, Material).
//
// We deliberately use a deny-list of well-known non-component classes rather than
// an allow-list of known components: Unity defines hundreds of component class IDs
// (Light, Camera, Animator, AudioSource, ...) that this tool does not yet model.
// Allow-listing would flag those valid components as corruption. Unknown classes
// are left untouched here and instead surface as UNKNOWN_CLASS_ID warnings.
func referencesKnownNonComponentBlock(graphResult *core.Graph, fileID int64) bool {
	objects := graphResult.ObjectsByID[fileID]
	if len(objects) == 0 {
		return false
	}
	for _, object := range objects {
		if object == nil {
			continue
		}
		if !isKnownNonComponentClassID(object.ClassID) {
			return false
		}
	}
	return true
}

func isKnownNonComponentClassID(classID int) bool {
	switch classID {
	case 1, // GameObject
		21,   // Material
		28,   // Texture2D
		43,   // Mesh
		48,   // Shader
		1001: // PrefabInstance
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
