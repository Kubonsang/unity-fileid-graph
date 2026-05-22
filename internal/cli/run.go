package cli

import (
	"fmt"
	"io"
	"os"

	"unity-fileid-graph/internal/parser"
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
