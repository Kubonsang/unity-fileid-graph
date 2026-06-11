package graph

import (
	"strconv"
	"strings"

	"github.com/Kubonsang/unity-fileid-graph/pkg/core"
)

type bodyLine struct {
	Indent        int
	Key           string
	EffectiveKey  string
	RawValue      string
	EffectiveText string
}

func scanBodyLines(body string, fileID int64) ([]bodyLine, []core.Issue) {
	rawLines := strings.Split(body, "\n")
	lines := make([]bodyLine, 0, len(rawLines))
	issues := make([]core.Issue, 0)

	for _, rawLine := range rawLines {
		analysisLine := strings.TrimSuffix(rawLine, "\r")
		if analysisLine == "" {
			continue
		}

		indent, hasTab := leadingIndent(analysisLine)
		if hasTab {
			issues = append(issues, core.Issue{
				Code:    core.IssueTabIndent,
				FileID:  fileID,
				Message: "tab indentation is unsupported in v0.2 field scanning",
			})
		}

		content := analysisLine
		if indent < len(analysisLine) {
			content = analysisLine[indent:]
		} else {
			content = ""
		}
		if strings.TrimSpace(content) == "" {
			continue
		}

		line := bodyLine{
			Indent:        indent,
			EffectiveText: strings.TrimSpace(content),
		}

		switch {
		case content == "-":
			line.Key = "-"
			line.EffectiveKey = "-"
		case strings.HasPrefix(content, "- "):
			line.Key = "-"
			line.EffectiveKey = "-"
			line.RawValue = strings.TrimSpace(content[2:])
			line.EffectiveText = line.RawValue
		case strings.Contains(content, ":"):
			key, value, _ := strings.Cut(content, ":")
			line.Key = key
			line.EffectiveKey = strings.TrimSpace(key)
			line.RawValue = strings.TrimLeft(value, " ")
			if line.RawValue != "" {
				line.EffectiveText = strings.TrimSpace(line.RawValue)
			}
		}

		lines = append(lines, line)
	}

	return lines, issues
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

func parseNestedFileID(lines []bodyLine, index int) (int64, bool) {
	if index < 0 || index >= len(lines) {
		return 0, false
	}

	if fileID, ok := parseInlineFileID(lines[index].RawValue); ok {
		return fileID, true
	}
	if fileID, ok := parseInlineFileID(lines[index].EffectiveText); ok {
		return fileID, true
	}

	parentIndent := lines[index].Indent
	for i := index + 1; i < len(lines); i++ {
		if lines[i].Indent <= parentIndent {
			break
		}

		if lines[i].EffectiveKey == "fileID" {
			fileID, err := strconv.ParseInt(strings.TrimSpace(lines[i].RawValue), 10, 64)
			if err == nil {
				return fileID, true
			}
		}

		if fileID, ok := parseInlineFileID(lines[i].EffectiveText); ok {
			return fileID, true
		}
	}

	return 0, false
}

func parseChildFileIDList(lines []bodyLine, index int) ([]int64, bool) {
	if index < 0 || index >= len(lines) {
		return nil, false
	}

	parentIndent := lines[index].Indent
	children := make([]int64, 0)
	found := false
	directIndent := -1

	for i := index + 1; i < len(lines); i++ {
		if lines[i].Indent <= parentIndent {
			break
		}

		if directIndent == -1 {
			directIndent = lines[i].Indent
		}
		if lines[i].Indent != directIndent {
			continue
		}

		fileID, ok := parseInlineFileID(lines[i].EffectiveText)
		if !ok {
			fileID, ok = parseNestedFileID(lines, i)
		}
		if !ok {
			return nil, false
		}

		children = append(children, fileID)
		found = true
	}

	if !found {
		return nil, false
	}

	return children, true
}

func leadingIndent(line string) (int, bool) {
	indent := 0
	hasTab := false

	for _, r := range line {
		if r == ' ' {
			indent++
			continue
		}
		if r == '\t' {
			indent++
			hasTab = true
			continue
		}
		break
	}

	return indent, hasTab
}
