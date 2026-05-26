package roundtrip

import (
	"bytes"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

func AssembleLosslessCopy(result *core.ParseResult) []byte {
	var buf bytes.Buffer
	buf.WriteString(result.PreambleRaw)
	for _, block := range result.Blocks {
		buf.WriteString(block.HeaderRaw)
		buf.WriteString(block.BodyRaw)
	}
	buf.WriteString(result.TrailerRaw)
	return buf.Bytes()
}

func DetectLineEnding(input []byte) string {
	if bytes.Contains(input, []byte("\r\n")) {
		return "CRLF"
	}
	if bytes.Contains(input, []byte("\n")) {
		return "LF"
	}
	return "NONE"
}
