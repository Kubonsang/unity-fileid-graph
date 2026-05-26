package graph

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

var guidPattern = regexp.MustCompile(`guid:\s*([0-9a-fA-F]{32})`)
var typePattern = regexp.MustCompile(`type:\s*(-?\d+)`)

func extractGameObject(fileID int64, body string) (*core.GameObjectNode, []core.Issue) {
	lines, issues := scanBodyLines(body, fileID)
	node := &core.GameObjectNode{
		FileID:     fileID,
		Components: []int64{},
	}

	for i, line := range lines {
		switch line.EffectiveKey {
		case "m_Name":
			node.Name = strings.TrimSpace(line.RawValue)
		case "-":
			if !strings.HasPrefix(line.EffectiveText, "component:") {
				continue
			}

			valueText := strings.TrimSpace(strings.TrimPrefix(line.EffectiveText, "component:"))
			if value, ok := parseInlineFileID(valueText); ok {
				node.Components = append(node.Components, value)
				continue
			}
			if value, ok := parseNestedFileID(lines, i); ok {
				node.Components = append(node.Components, value)
				continue
			}

			issues = append(issues, core.Issue{
				Code:    core.IssueUnknownFieldShape,
				FileID:  fileID,
				Message: "unsupported GameObject.m_Component entry shape",
			})
		}
	}

	return node, issues
}

func extractComponentRef(fileID int64, body string, classID int, typeName string) (*core.ComponentNode, []bodyLine, []core.Issue) {
	lines, issues := scanBodyLines(body, fileID)
	component := &core.ComponentNode{
		FileID:   fileID,
		ClassID:  classID,
		TypeName: typeName,
	}

	for i, line := range lines {
		if line.EffectiveKey != "m_GameObject" {
			continue
		}

		if value, ok := parseInlineFileID(line.RawValue); ok {
			component.GameObject = value
			component.HasGameObject = true
			return component, lines, issues
		}
		if value, ok := parseNestedFileID(lines, i); ok {
			component.GameObject = value
			component.HasGameObject = true
			return component, lines, issues
		}

		issues = append(issues, core.Issue{
			Code:    core.IssueUnknownFieldShape,
			FileID:  fileID,
			Message: "unsupported Component.m_GameObject shape",
		})
	}

	return component, lines, issues
}

func extractTransform(fileID int64, body string) (*core.ComponentNode, *core.TransformNode, []core.Issue) {
	component, lines, issues := extractComponentRef(fileID, body, 4, "Transform")
	transform := &core.TransformNode{
		FileID:     fileID,
		GameObject: component.GameObject,
		Children:   []int64{},
	}

	for i, line := range lines {
		switch line.EffectiveKey {
		case "m_Father":
			if value, ok := parseInlineFileID(line.RawValue); ok {
				transform.Father = value
				continue
			}
			if value, ok := parseNestedFileID(lines, i); ok {
				transform.Father = value
				continue
			}

			issues = append(issues, core.Issue{
				Code:    core.IssueUnknownFieldShape,
				FileID:  fileID,
				Message: "unsupported Transform.m_Father shape",
			})
		case "m_Children":
			if strings.TrimSpace(line.RawValue) == "[]" {
				transform.Children = []int64{}
				continue
			}
			if children, ok := parseChildFileIDList(lines, i); ok {
				transform.Children = children
				continue
			}
			issues = append(issues, core.Issue{
				Code:    core.IssueUnknownFieldShape,
				FileID:  fileID,
				Message: "unsupported Transform.m_Children shape",
			})
		}
	}

	return component, transform, issues
}

func extractMonoBehaviour(fileID int64, body string) (*core.ComponentNode, []core.Issue) {
	component, lines, issues := extractComponentRef(fileID, body, 114, "MonoBehaviour")

	for _, line := range lines {
		if line.EffectiveKey != "m_Script" {
			continue
		}

		scriptFileID, ok := parseInlineFileID(line.RawValue)
		if !ok {
			issues = append(issues, core.Issue{
				Code:    core.IssueUnknownFieldShape,
				FileID:  fileID,
				Message: "unsupported MonoBehaviour.m_Script shape",
			})
			continue
		}

		guidMatch := guidPattern.FindStringSubmatch(line.RawValue)
		typeMatch := typePattern.FindStringSubmatch(line.RawValue)
		if guidMatch == nil || typeMatch == nil {
			issues = append(issues, core.Issue{
				Code:    core.IssueUnknownFieldShape,
				FileID:  fileID,
				Message: "unsupported MonoBehaviour.m_Script shape",
			})
			continue
		}

		typeValue, err := strconv.Atoi(typeMatch[1])
		if err != nil {
			issues = append(issues, core.Issue{
				Code:    core.IssueUnknownFieldShape,
				FileID:  fileID,
				Message: "unsupported MonoBehaviour.m_Script shape",
			})
			continue
		}

		component.Script = &core.ScriptRef{
			FileID: scriptFileID,
			GUID:   guidMatch[1],
			Type:   typeValue,
		}
	}

	return component, issues
}

func extractGenericComponent(fileID int64, body string, classID int, typeName string) (*core.ComponentNode, []core.Issue) {
	component, _, issues := extractComponentRef(fileID, body, classID, typeName)
	return component, issues
}
