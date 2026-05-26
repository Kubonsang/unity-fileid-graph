package cli

import (
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Kubonsang/unity-fileid-graph/internal/check"
	"github.com/Kubonsang/unity-fileid-graph/internal/core"
	"github.com/Kubonsang/unity-fileid-graph/internal/graph"
	"github.com/Kubonsang/unity-fileid-graph/internal/mutate"
	"github.com/Kubonsang/unity-fileid-graph/internal/parser"
	"github.com/Kubonsang/unity-fileid-graph/internal/roundtrip"
)

var validNamespaces = map[string]struct{}{
	"prefab": {},
	"scene":  {},
	"asset":  {},
	"mat":    {},
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) < 3 {
		writeUsage(stderr)
		return 2
	}

	if _, ok := validNamespaces[args[0]]; !ok {
		writeUsage(stderr)
		return 2
	}

	command := args[1]
	if command != "blocks" && command != "graph" && command != "check" && command != "roundtrip" && command != "set" {
		writeUsage(stderr)
		return 2
	}

	if command == "roundtrip" {
		return runRoundtrip(args, stdout, stderr)
	}
	if command == "set" {
		return runSet(args, stdout, stderr)
	}

	if len(args) != 3 {
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

	if args[1] == "blocks" {
		return writeBlocks(stdout, result)
	}

	graphResult, err := graph.Build(result)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "graph %s: %v\n", args[2], err)
		return 1
	}

	if args[1] == "graph" {
		return writeGraph(stdout, graphResult)
	}

	return writeCheck(stdout, check.Run(graphResult))
}

func writeBlocks(stdout io.Writer, result *core.ParseResult) int {
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
	_, _ = fmt.Fprintln(stderr, "usage: uyaml <prefab|scene|asset|mat> <blocks|graph|check> <file>")
	_, _ = fmt.Fprintln(stderr, "   or: uyaml <prefab|scene|asset|mat> roundtrip <file> --out <dest> [--mode lossless-block-copy]")
	_, _ = fmt.Fprintln(stderr, "   or: uyaml <prefab|scene|asset|mat> set <file> --id <fileID> --field <top-level-field-name> --value <value>")
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
		if len(graphResult.Transforms) > 0 {
			_, _ = fmt.Fprintln(stdout)
		}
		for _, issue := range graphResult.Issues {
			_, _ = fmt.Fprintf(stdout, "WARN code=%s file_id=%d message=%q\n", issue.Code, issue.FileID, issue.Message)
		}
	}

	return 0
}

func writeCheck(stdout io.Writer, result *core.CheckResult) int {
	_, _ = fmt.Fprintf(stdout, "GRAPH_CHECK status=%s blocks=%d gameobjects=%d components=%d transforms=%d\n",
		result.Status,
		result.BlockCount,
		result.GameObjectCount,
		result.ComponentCount,
		result.TransformCount,
	)

	for _, finding := range result.Errors {
		_, _ = fmt.Fprintf(stdout, "ERROR code=%s", finding.Code)
		writeCheckErrorFields(stdout, finding)
		_, _ = fmt.Fprintln(stdout)
	}

	for _, finding := range result.Warnings {
		_, _ = fmt.Fprintf(stdout, "WARN code=%s", finding.Code)
		writeCheckWarningFields(stdout, finding)
		_, _ = fmt.Fprintln(stdout)
	}

	if result.Status == core.CheckStatusError {
		return 1
	}
	return 0
}

func writeCheckErrorFields(stdout io.Writer, finding core.CheckFinding) {
	switch finding.Code {
	case core.CheckDuplicateFileID:
		_, _ = fmt.Fprintf(stdout, " file_id=%d duplicates=%d", finding.FileID, finding.DuplicateCount)
	case core.CheckMissingComponentBlock:
		_, _ = fmt.Fprintf(stdout, " go=%d component_id=%d reason=%s", finding.GameObjectID, finding.ComponentID, finding.Reason)
	case core.CheckMissingGameObjectBlock:
		_, _ = fmt.Fprintf(stdout, " component=%d m_GameObject=%d reason=%s", finding.ComponentID, finding.GameObjectID, finding.Reason)
	case core.CheckGoComponentBackrefMismatch:
		_, _ = fmt.Fprintf(stdout, " component=%d go=%d reason=%s", finding.ComponentID, finding.GameObjectID, finding.Reason)
	case core.CheckTransformParentChildMismatch:
		if finding.ParentID != 0 {
			_, _ = fmt.Fprintf(stdout, " parent=%d", finding.ParentID)
		}
		if finding.ChildID != 0 {
			_, _ = fmt.Fprintf(stdout, " child=%d", finding.ChildID)
		}
		if finding.TransformID != 0 {
			_, _ = fmt.Fprintf(stdout, " transform=%d", finding.TransformID)
		}
		if finding.Reason != "" {
			_, _ = fmt.Fprintf(stdout, " reason=%s", finding.Reason)
		}
	case core.CheckMissingTransformComponent:
		_, _ = fmt.Fprintf(stdout, " go=%d reason=%s", finding.GameObjectID, finding.Reason)
	case core.CheckSuspiciousMonoBehaviourScript:
		_, _ = fmt.Fprintf(stdout, " component=%d reason=%s", finding.ComponentID, finding.Reason)
	default:
		if finding.FileID != 0 {
			_, _ = fmt.Fprintf(stdout, " file_id=%d", finding.FileID)
		}
		if finding.Reason != "" {
			_, _ = fmt.Fprintf(stdout, " reason=%s", finding.Reason)
		}
	}
}

func writeCheckWarningFields(stdout io.Writer, finding core.CheckFinding) {
	if finding.FileID != 0 {
		_, _ = fmt.Fprintf(stdout, " file_id=%d", finding.FileID)
	}
	if finding.ComponentID != 0 {
		_, _ = fmt.Fprintf(stdout, " component=%d", finding.ComponentID)
	}
	if finding.Reason != "" {
		_, _ = fmt.Fprintf(stdout, " reason=%s", finding.Reason)
	}
	if finding.Message != "" {
		_, _ = fmt.Fprintf(stdout, " message=%q", finding.Message)
	}
}

func runRoundtrip(args []string, stdout, stderr io.Writer) int {
	if len(args) != 5 && len(args) != 7 {
		writeUsage(stderr)
		return 2
	}

	inputPath := args[2]
	outputPath := ""
	mode := core.RoundtripModeLosslessBlockCopy

	for i := 3; i < len(args); i += 2 {
		if i+1 >= len(args) {
			writeUsage(stderr)
			return 2
		}
		switch args[i] {
		case "--out":
			outputPath = args[i+1]
		case "--mode":
			mode = args[i+1]
		default:
			writeUsage(stderr)
			return 2
		}
	}

	if outputPath == "" || mode != core.RoundtripModeLosslessBlockCopy {
		writeUsage(stderr)
		return 2
	}

	input, err := os.ReadFile(inputPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "read %s: %v\n", inputPath, err)
		return 1
	}

	result, err := roundtrip.RunLosslessCopy(input, outputPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "roundtrip %s: %v\n", inputPath, err)
		return 1
	}

	return writeRoundtrip(stdout, result)
}

func writeRoundtrip(stdout io.Writer, result *core.RoundtripResult) int {
	_, _ = fmt.Fprintf(stdout,
		"ROUNDTRIP status=%s mode=%s bytes_equal=%d reparsed=%d block_sequence_equal=%d graph_check=%s line_endings=%s editor_open=%s out=%s\n",
		result.Status,
		result.Mode,
		boolInt(result.BytesEqual),
		boolInt(result.Reparsed),
		boolInt(result.BlockSequenceEqual),
		result.GraphCheckStatus,
		result.LineEndingStyle,
		result.EditorOpenStatus,
		result.OutputPath,
	)

	if result.Status == core.RoundtripStatusError {
		return 1
	}
	return 0
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func runSet(args []string, stdout, stderr io.Writer) int {
	if len(args) != 9 {
		writeUsage(stderr)
		return 2
	}

	opts := core.SetOptions{InputPath: args[2]}
	idCount := 0
	fieldCount := 0
	valueCount := 0
	for i := 3; i < len(args); i += 2 {
		if i+1 >= len(args) {
			writeUsage(stderr)
			return 2
		}
		switch args[i] {
		case "--id":
			idCount++
			if idCount > 1 {
				writeUsage(stderr)
				return 2
			}
			fileID, err := strconv.ParseInt(args[i+1], 10, 64)
			if err != nil {
				writeUsage(stderr)
				return 2
			}
			opts.FileID = fileID
		case "--field":
			fieldCount++
			if fieldCount > 1 {
				writeUsage(stderr)
				return 2
			}
			opts.Field = args[i+1]
		case "--value":
			valueCount++
			if valueCount > 1 {
				writeUsage(stderr)
				return 2
			}
			opts.Value = args[i+1]
		default:
			writeUsage(stderr)
			return 2
		}
	}
	if idCount != 1 || fieldCount != 1 || valueCount != 1 || opts.Field == "" {
		writeUsage(stderr)
		return 2
	}

	result, err := mutate.RunSet(opts)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "set %s: %v\n", opts.InputPath, err)
		return 1
	}
	return writeSet(stdout, result)
}

func writeSet(stdout io.Writer, result *core.SetResult) int {
	if result.Status == core.MutationStatusBlocked {
		_, _ = fmt.Fprintf(stdout, "SET status=BLOCKED code=%s file_id=%d field=%s message=%q\n", result.Code, result.FileID, result.Field, result.Message)
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "SET status=%s file_id=%d field=%s old=%s new=%s pre_check=%s temp_check=%s final_check=%s backup=%s\n",
		result.Status,
		result.FileID,
		result.Field,
		result.OldValue,
		result.NewValue,
		result.PreCheck,
		result.TempCheck,
		result.FinalCheck,
		result.BackupPath,
	)

	if result.Status == core.MutationStatusError {
		return 1
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
