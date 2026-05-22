package cli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
	"github.com/Kubonsang/unity-fileid-graph/internal/parser"
)

var validNamespaces = map[string]struct{}{
	"prefab": {},
	"scene":  {},
	"asset":  {},
	"mat":    {},
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 3 {
		writeUsage(stderr)
		return 2
	}

	if _, ok := validNamespaces[args[0]]; !ok {
		writeUsage(stderr)
		return 2
	}

	if args[1] != "blocks" {
		writeUsage(stderr)
		return 2
	}

	input, err := os.ReadFile(args[2])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read %s: %v\n", args[2], err)
		return 1
	}

	result, err := parser.Parse(input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "parse %s: %v\n", args[2], err)
		return 1
	}

	for _, block := range result.Blocks {
		stripped := 0
		if block.IsStripped {
			stripped = 1
		}
		_, _ = fmt.Fprintf(stdout, "BLOCK index=%d class_id=%d file_id=%d stripped=%d\n", block.Index, block.ClassID, block.FileID, stripped)
	}
	return 0
}

func writeUsage(stderr io.Writer) {
	_, _ = fmt.Fprintln(stderr, "usage: uyaml <prefab|scene|asset|mat> blocks <file>")
}

func writeGraph(stdout io.Writer, graphResult *core.Graph) int {
	for _, goID := range sortedGameObjectKeys(graphResult.GameObjects) {
		goNode := graphResult.GameObjects[goID]
		_, _ = fmt.Fprintf(stdout, "GAMEOBJECT id=%d name=%s\n", goNode.FileID, goNode.Name)
		for _, componentID := range goNode.Components {
			component, ok := graphResult.Components[componentID]
			if !ok || component == nil {
				_, _ = fmt.Fprintf(stdout, "  component=%d type=UNKNOWN\n", componentID)
				continue
			}
			_, _ = fmt.Fprintf(stdout, "  component=%d type=%s\n", componentID, component.TypeName)
		}
		_, _ = fmt.Fprintln(stdout)
	}

	for _, componentID := range sortedComponentKeys(graphResult.Components) {
		component := graphResult.Components[componentID]
		gameObjectText := "UNKNOWN"
		if component.HasGameObject {
			gameObjectText = strconv.FormatInt(component.GameObject, 10)
		}
		_, _ = fmt.Fprintf(stdout, "COMPONENT id=%d type=%s game_object=%s\n", component.FileID, component.TypeName, gameObjectText)
		if component.Script != nil {
			_, _ = fmt.Fprintf(stdout, "  script_file_id=%d guid=%s type=%d\n", component.Script.FileID, component.Script.GUID, component.Script.Type)
		}
		_, _ = fmt.Fprintln(stdout)
	}

	for _, transformID := range sortedTransformKeys(graphResult.Transforms) {
		transform := graphResult.Transforms[transformID]
		children := "none"
		if len(transform.Children) > 0 {
			childParts := make([]string, 0, len(transform.Children))
			for _, child := range transform.Children {
				childParts = append(childParts, strconv.FormatInt(child, 10))
			}
			children = strings.Join(childParts, ",")
		}
		_, _ = fmt.Fprintf(stdout, "TRANSFORM id=%d game_object=%d father=%d children=%s\n", transform.FileID, transform.GameObject, transform.Father, children)
	}

	if len(graphResult.Issues) > 0 {
		_, _ = fmt.Fprintln(stdout)
		for _, issue := range graphResult.Issues {
			_, _ = fmt.Fprintf(stdout, "WARN code=%s file_id=%d message=%q\n", issue.Code, issue.FileID, issue.Message)
		}
	}

	return 0
}

func sortedGameObjectKeys(gameObjects map[int64]*core.GameObjectNode) []int64 {
	keys := make([]int64, 0, len(gameObjects))
	for key := range gameObjects {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func sortedComponentKeys(components map[int64]*core.ComponentNode) []int64 {
	keys := make([]int64, 0, len(components))
	for key := range components {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func sortedTransformKeys(transforms map[int64]*core.TransformNode) []int64 {
	keys := make([]int64, 0, len(transforms))
	for key := range transforms {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}
