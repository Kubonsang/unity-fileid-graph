package refs

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

var inlinePPtrPattern = regexp.MustCompile(`\{fileID:\s*(-?\d+)(?:,\s*guid:\s*([0-9a-fA-F]{32}))?(?:,\s*type:\s*(-?\d+))?\}`)

func Extract(parsed *core.ParseResult, namespace string, file string) *core.RefsResult {
	result := &core.RefsResult{
		Namespace:  namespace,
		File:       file,
		References: []core.Reference{},
		Issues:     []core.Issue{},
	}
	if parsed == nil {
		result.RecomputeStatus()
		return result
	}

	for _, block := range parsed.Blocks {
		typeName := typeNameForClassID(block.ClassID)
		references, issues := extractBlockRefs(block, typeName)
		result.References = append(result.References, references...)
		result.Issues = append(result.Issues, issues...)
	}

	result.RecomputeStatus()
	return result
}

func extractBlockRefs(block *core.Block, typeName string) ([]core.Reference, []core.Issue) {
	lines := strings.Split(block.BodyRaw, "\n")
	references := []core.Reference{}
	issues := []core.Issue{}
	listIndexByField := map[string]int{}
	currentListField := ""

	for _, raw := range lines {
		line := strings.TrimSuffix(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		field := ""
		value := ""
		if strings.HasPrefix(trimmed, "- ") {
			if currentListField == "" {
				continue
			}
			itemText := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			field = fieldPathForListItem(currentListField, listIndexByField)
			value = itemText
		} else {
			key, rawValue, ok := strings.Cut(trimmed, ":")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(rawValue)
			if value == "" {
				listIndexByField[key] = 0
				currentListField = key
				continue
			}
			currentListField = ""
			field = key
		}

		if skipGraphStructuralField(field) {
			continue
		}
		matches := inlinePPtrPattern.FindAllStringSubmatch(value, -1)
		for _, match := range matches {
			fileID, err := strconv.ParseInt(match[1], 10, 64)
			if err != nil {
				issues = append(issues, core.Issue{Code: core.IssueUnknownFieldShape, FileID: block.FileID, Message: "unsupported PPtr fileID"})
				continue
			}
			reference := core.Reference{
				BlockFileID: block.FileID,
				ClassID:     block.ClassID,
				TypeName:    typeName,
				Field:       field,
				FileID:      fileID,
				RawValue:    strings.TrimSpace(match[0]),
			}
			if match[2] != "" {
				reference.GUID = match[2]
				reference.HasGUID = true
			}
			if match[3] != "" {
				typeValue, err := strconv.Atoi(match[3])
				if err != nil {
					issues = append(issues, core.Issue{Code: core.IssueUnknownFieldShape, FileID: block.FileID, Message: "unsupported PPtr type"})
					continue
				}
				reference.Type = typeValue
				reference.HasType = true
			}
			references = append(references, reference)
		}
	}

	return references, issues
}

func fieldPathForListItem(field string, indexes map[string]int) string {
	index := indexes[field]
	indexes[field] = index + 1
	if field == "m_Component" {
		return "m_Component[" + strconv.Itoa(index) + "].component"
	}
	return field + "[" + strconv.Itoa(index) + "]"
}

func skipGraphStructuralField(field string) bool {
	return field == "m_GameObject" || field == "m_Father"
}

func typeNameForClassID(classID int) string {
	switch classID {
	case 1:
		return "GameObject"
	case 4:
		return "Transform"
	case 23:
		return "MeshRenderer"
	case 33:
		return "MeshFilter"
	case 54:
		return "Rigidbody"
	case 65:
		return "BoxCollider"
	case 114:
		return "MonoBehaviour"
	default:
		return "UNKNOWN"
	}
}
