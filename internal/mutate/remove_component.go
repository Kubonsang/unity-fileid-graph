package mutate

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Kubonsang/unity-fileid-graph/pkg/core"
)

var componentRefPattern = regexp.MustCompile(`^  - component: \{fileID:\s*(-?\d+)\}\s*$`)

var ErrComponentRefNotFound = errors.New("component ref not found")

type componentEntry struct {
	LineIndex int
	FileID    int64
}

func ExtractComponentOwnerGameObject(body string) (int64, error) {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "m_GameObject:") {
			continue
		}
		valueText := strings.TrimSpace(strings.TrimPrefix(trimmed, "m_GameObject:"))
		id, ok := parseInlineFileID(valueText)
		if !ok || id == 0 {
			return 0, fmt.Errorf("component owner GameObject missing or unsupported")
		}
		return id, nil
	}
	return 0, fmt.Errorf("component owner GameObject missing or unsupported")
}

func RemoveComponentEntry(body string, targetID int64) (string, error) {
	entries, err := scanExactComponentEntries(body)
	if err != nil {
		return "", err
	}

	matchIndex := -1
	for _, entry := range entries {
		if entry.FileID != targetID {
			continue
		}
		if matchIndex != -1 {
			return "", fmt.Errorf("duplicate component ref")
		}
		matchIndex = entry.LineIndex
	}

	if matchIndex == -1 {
		return "", ErrComponentRefNotFound
	}

	lines := strings.SplitAfter(body, "\n")
	lines = append(lines[:matchIndex], lines[matchIndex+1:]...)
	return strings.Join(lines, ""), nil
}

func HasExactComponentEntry(body string, targetID int64) (bool, error) {
	entries, err := scanExactComponentEntries(body)
	if err != nil {
		return false, err
	}
	for _, entry := range entries {
		if entry.FileID == targetID {
			return true, nil
		}
	}
	return false, nil
}

func scanExactComponentEntries(body string) ([]componentEntry, error) {
	lines := strings.SplitAfter(body, "\n")
	inComponentList := false
	entries := []componentEntry{}

	for i, line := range lines {
		lineNoNL := strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		indent := leadingSpaces(lineNoNL)
		trimmed := strings.TrimSpace(lineNoNL)

		if indent == 2 && trimmed == "m_Component:" {
			inComponentList = true
			continue
		}
		if !inComponentList {
			continue
		}
		if indent < 2 {
			break
		}
		if indent == 2 && !strings.HasPrefix(trimmed, "- ") {
			break
		}
		if indent != 2 {
			return nil, ErrUnsupportedFieldShape
		}
		if !componentRefPattern.MatchString(lineNoNL) {
			if strings.HasPrefix(trimmed, "- ") {
				return nil, ErrUnsupportedFieldShape
			}
			continue
		}

		valueText := strings.TrimSpace(strings.TrimPrefix(trimmed, "- component:"))
		id, ok := parseInlineFileID(valueText)
		if !ok {
			return nil, ErrUnsupportedFieldShape
		}
		entries = append(entries, componentEntry{LineIndex: i, FileID: id})
	}

	return entries, nil
}

func parseInlineFileID(raw string) (int64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	if raw == "-" {
		return 0, false
	}
	if strings.HasPrefix(raw, "- ") {
		raw = strings.TrimSpace(raw[2:])
	}
	if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		raw = strings.TrimSpace(raw[1 : len(raw)-1])
	}

	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "fileID:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(part, "fileID:"))
		fileID, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, false
		}
		return fileID, true
	}

	return 0, false
}

func DropBlockByFileID(parsed *core.ParseResult, fileID int64) (*core.ParseResult, error) {
	blocks := make([]*core.Block, 0, len(parsed.Blocks)-1)
	found := false
	for _, block := range parsed.Blocks {
		if block.FileID == fileID {
			if found {
				return nil, fmt.Errorf("duplicate block match")
			}
			found = true
			continue
		}
		blocks = append(blocks, block)
	}
	if !found {
		return nil, fmt.Errorf("target block not found")
	}
	return &core.ParseResult{
		Blocks:      blocks,
		PreambleRaw: parsed.PreambleRaw,
		TrailerRaw:  parsed.TrailerRaw,
	}, nil
}

func remainingBlocksContainFileIDReference(parsed *core.ParseResult, fileID int64) bool {
	if parsed == nil {
		return false
	}

	pattern := regexp.MustCompile(`\bfileID:\s*` + regexp.QuoteMeta(strconv.FormatInt(fileID, 10)) + `\b`)
	for _, block := range parsed.Blocks {
		if block == nil {
			continue
		}
		if pattern.MatchString(block.BodyRaw) {
			return true
		}
	}
	return false
}
