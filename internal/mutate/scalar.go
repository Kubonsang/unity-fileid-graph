package mutate

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

var ErrFieldNotFound = errors.New("field not found")
var ErrUnsupportedFieldShape = errors.New("unsupported field shape")

type ScalarEditPlan struct {
	Field    string
	OldValue string
	NewValue string
	NewBody  string
}

func PlanScalarEdit(body, field, rawValue string) (*ScalarEditPlan, error) {
	lines := strings.SplitAfter(body, "\n")
	for i, line := range lines {
		lineNoNL := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if lineNoNL == "" {
			continue
		}
		if leadingSpaces(lineNoNL) != 2 {
			continue
		}

		trimmed := strings.TrimLeft(lineNoNL, " ")
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok || key != field {
			continue
		}
		if value == "" || !strings.HasPrefix(value, " ") {
			return nil, ErrUnsupportedFieldShape
		}

		oldValue := strings.TrimSpace(value)
		if oldValue == "" {
			return nil, ErrUnsupportedFieldShape
		}
		if strings.HasPrefix(oldValue, "{") || strings.HasPrefix(oldValue, "[") || oldValue == "|" || oldValue == ">" {
			return nil, ErrUnsupportedFieldShape
		}

		newValue, err := formatScalarReplacement(field, oldValue, rawValue)
		if err != nil {
			return nil, err
		}

		newline := ""
		if strings.HasSuffix(line, "\r\n") {
			newline = "\r\n"
		} else if strings.HasSuffix(line, "\n") {
			newline = "\n"
		}

		lines[i] = "  " + field + ": " + newValue + newline
		return &ScalarEditPlan{
			Field:    field,
			OldValue: oldValue,
			NewValue: newValue,
			NewBody:  strings.Join(lines, ""),
		}, nil
	}

	return nil, ErrFieldNotFound
}

func ValidateBlockMutation(block *core.Block) (string, string) {
	if block.IsStripped {
		return core.MutationCodeStrippedObjectBlocked, "native scalar writes to stripped objects are blocked in v0.5"
	}
	switch block.ClassID {
	case 114:
		return core.MutationCodeMonoBehaviourWriteBlocked, "native scalar writes to MonoBehaviour are blocked in v0.5"
	default:
		return "", ""
	}
}

func FindUniqueBlockByFileID(parsed *core.ParseResult, fileID int64) (*core.Block, string) {
	var match *core.Block
	count := 0
	for _, block := range parsed.Blocks {
		if block.FileID != fileID {
			continue
		}
		count++
		if count == 1 {
			match = block
		}
	}
	switch count {
	case 0:
		return nil, core.MutationCodeFileIDNotFound
	case 1:
		return match, ""
	default:
		return nil, core.MutationCodeDuplicateFileID
	}
}

func leadingSpaces(line string) int {
	count := 0
	for count < len(line) && line[count] == ' ' {
		count++
	}
	return count
}

func formatScalarReplacement(field, oldValue, rawValue string) (string, error) {
	if isKnownBoolField(field) {
		switch rawValue {
		case "0", "1", "true", "false":
			return rawValue, nil
		default:
			return "", fmt.Errorf("invalid bool literal")
		}
	}
	if _, err := strconv.ParseInt(oldValue, 10, 64); err == nil {
		if _, err := strconv.ParseInt(rawValue, 10, 64); err != nil {
			return "", err
		}
		return rawValue, nil
	}
	if _, err := strconv.ParseFloat(oldValue, 64); err == nil {
		if _, err := strconv.ParseFloat(rawValue, 64); err != nil {
			return "", err
		}
		return rawValue, nil
	}
	return strconv.Quote(rawValue), nil
}

func isKnownBoolField(field string) bool {
	switch field {
	case "m_IsActive", "m_Enabled":
		return true
	default:
		return false
	}
}
