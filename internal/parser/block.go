package parser

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

type headerMeta struct {
	ClassID    int
	FileID     int64
	IsStripped bool
}

var headerPattern = regexp.MustCompile(`^--- !u!(\d+) &(\d+)( stripped)?(?:\r?\n)$`)

func parseHeader(header string) (headerMeta, error) {
	matches := headerPattern.FindStringSubmatch(header)
	if matches == nil {
		return headerMeta{}, fmt.Errorf("invalid Unity header: %q", header)
	}

	classID, err := strconv.Atoi(matches[1])
	if err != nil {
		return headerMeta{}, fmt.Errorf("parse class id: %w", err)
	}

	fileID, err := strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return headerMeta{}, fmt.Errorf("parse file id: %w", err)
	}

	return headerMeta{
		ClassID:    classID,
		FileID:     fileID,
		IsStripped: matches[3] == " stripped",
	}, nil
}

func Parse(input []byte) (*core.ParseResult, error) {
	result := &core.ParseResult{}
	headerStarts, invalidHeader := findHeaderLineStarts(input)
	if invalidHeader != "" {
		return nil, fmt.Errorf("invalid Unity header: %q", invalidHeader)
	}
	if len(headerStarts) == 0 {
		result.PreambleRaw = string(input)
		return result, nil
	}

	result.PreambleRaw = string(input[:headerStarts[0]])
	result.Blocks = make([]*core.Block, 0, len(headerStarts))

	for i, headerStart := range headerStarts {
		headerEnd := nextLineEnd(input, headerStart)
		bodyEnd := len(input)
		if i+1 < len(headerStarts) {
			bodyEnd = headerStarts[i+1]
		}

		headerRaw := string(input[headerStart:headerEnd])
		meta, err := parseHeader(headerRaw)
		if err != nil {
			return nil, err
		}

		result.Blocks = append(result.Blocks, &core.Block{
			Index:      i,
			ClassID:    meta.ClassID,
			FileID:     meta.FileID,
			HeaderRaw:  headerRaw,
			BodyRaw:    string(input[headerEnd:bodyEnd]),
			IsStripped: meta.IsStripped,
		})
	}

	moveTrailingDocumentMarker(result)
	return result, nil
}

func findHeaderLineStarts(input []byte) ([]int, string) {
	var starts []int
	var invalidHeader string
	lineStart := 0

	for lineStart < len(input) {
		lineEnd := nextLineEnd(input, lineStart)
		line := string(input[lineStart:lineEnd])
		if len(line) > 0 && input[lineStart] == '-' {
			if _, err := parseHeader(line); err == nil {
				starts = append(starts, lineStart)
			} else if strings.HasPrefix(line, "--- !u!") && invalidHeader == "" {
				invalidHeader = line
			}
		}
		lineStart = lineEnd
	}

	return starts, invalidHeader
}

func nextLineEnd(input []byte, start int) int {
	for i := start; i < len(input); i++ {
		if input[i] == '\n' {
			return i + 1
		}
	}
	return len(input)
}

func moveTrailingDocumentMarker(result *core.ParseResult) {
	if len(result.Blocks) == 0 {
		return
	}

	lastBlock := result.Blocks[len(result.Blocks)-1]
	body := []byte(lastBlock.BodyRaw)

	for _, marker := range [][]byte{[]byte("...\r\n"), []byte("...\n"), []byte("...")} {
		if bytes.HasSuffix(body, marker) {
			lastBlock.BodyRaw = string(body[:len(body)-len(marker)])
			result.TrailerRaw = string(marker)
			return
		}
	}
}
