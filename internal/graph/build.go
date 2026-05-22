package graph

import "github.com/Kubonsang/unity-fileid-graph/internal/core"

func Build(parsed *core.ParseResult) (*core.Graph, error) {
	graph := &core.Graph{
		Blocks:      []*core.Block{},
		BlocksByID:  map[int64][]*core.Block{},
		ObjectsByID: map[int64][]*core.UnityObject{},
		GameObjects: map[int64]*core.GameObjectNode{},
		Components:  map[int64]*core.ComponentNode{},
		Transforms:  map[int64]*core.TransformNode{},
		Issues:      []core.Issue{},
	}

	if parsed == nil {
		return graph, nil
	}

	for _, block := range parsed.Blocks {
		graph.Blocks = append(graph.Blocks, block)
		graph.BlocksByID[block.FileID] = append(graph.BlocksByID[block.FileID], block)

		object := &core.UnityObject{
			FileID:   block.FileID,
			ClassID:  block.ClassID,
			TypeName: typeNameForClassID(block.ClassID),
			Block:    block,
		}
		graph.ObjectsByID[block.FileID] = append(graph.ObjectsByID[block.FileID], object)

		switch block.ClassID {
		case 1:
			node, issues := extractGameObject(block.FileID, block.BodyRaw)
			graph.GameObjects[block.FileID] = node
			graph.Issues = append(graph.Issues, issues...)
		case 4:
			component, transform, issues := extractTransform(block.FileID, block.BodyRaw)
			graph.Components[block.FileID] = component
			graph.Transforms[block.FileID] = transform
			graph.Issues = append(graph.Issues, issues...)
		case 114:
			component, issues := extractMonoBehaviour(block.FileID, block.BodyRaw)
			graph.Components[block.FileID] = component
			graph.Issues = append(graph.Issues, issues...)
		default:
			graph.Issues = append(graph.Issues, core.Issue{
				Code:    core.IssueUnknownClassID,
				FileID:  block.FileID,
				Message: "graph build skipped unsupported class id",
			})
		}
	}

	for _, goNode := range graph.GameObjects {
		for _, componentID := range goNode.Components {
			transform, ok := graph.Transforms[componentID]
			if !ok || transform == nil {
				continue
			}

			goNode.Transform = componentID
			break
		}
	}

	return graph, nil
}

func typeNameForClassID(classID int) string {
	switch classID {
	case 1:
		return "GameObject"
	case 4:
		return "Transform"
	case 114:
		return "MonoBehaviour"
	default:
		return "UNKNOWN"
	}
}
