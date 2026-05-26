package mutate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Kubonsang/unity-fileid-graph/internal/check"
	"github.com/Kubonsang/unity-fileid-graph/internal/core"
	"github.com/Kubonsang/unity-fileid-graph/internal/graph"
	"github.com/Kubonsang/unity-fileid-graph/internal/parser"
	"github.com/Kubonsang/unity-fileid-graph/internal/roundtrip"
)

type fileOps struct {
	Rename func(oldpath, newpath string) error
	Remove func(path string) error
}

func RunSet(opts core.SetOptions) (*core.SetResult, error) {
	ops := fileOps{
		Rename: os.Rename,
		Remove: os.Remove,
	}
	return runSetWithFileOps(opts, ops)
}

func runSetWithFileOps(opts core.SetOptions, ops fileOps) (*core.SetResult, error) {
	input, err := os.ReadFile(opts.InputPath)
	if err != nil {
		return nil, err
	}

	parsed, err := parser.Parse(input)
	if err != nil {
		return nil, err
	}

	graphResult, err := graph.Build(parsed)
	if err != nil {
		return nil, err
	}
	pre := check.Run(graphResult)

	result := &core.SetResult{FileID: opts.FileID, Field: opts.Field, PreCheck: pre.Status}

	block, code := FindUniqueBlockByFileID(parsed, opts.FileID)
	if code != "" {
		result.Code = code
		result.MarkBlocked("mutation target is missing or ambiguous")
		return result, nil
	}
	if code, msg := ValidateBlockMutation(block); code != "" {
		result.Code = code
		result.MarkBlocked(msg)
		return result, nil
	}
	if pre.Status == core.CheckStatusError {
		result.Code = core.MutationCodePrecheckError
		result.MarkBlocked("graph-check before mutation returned ERROR")
		return result, nil
	}

	edit, err := PlanScalarEdit(block.BodyRaw, opts.Field, opts.Value)
	if err != nil {
		switch {
		case errors.Is(err, ErrFieldNotFound):
			result.Code = core.MutationCodeFieldNotFound
		case errors.Is(err, ErrUnsupportedFieldShape):
			result.Code = core.MutationCodeUnsupportedFieldShape
		default:
			result.Code = core.MutationCodeUnsupportedFieldShape
		}
		result.MarkBlocked(err.Error())
		return result, nil
	}
	result.OldValue = edit.OldValue
	result.NewValue = edit.NewValue
	block.BodyRaw = edit.NewBody

	output := roundtrip.AssembleLosslessCopy(parsed)
	dir := filepath.Dir(opts.InputPath)
	tempFile, err := os.CreateTemp(dir, filepath.Base(opts.InputPath)+".tmp-*")
	if err != nil {
		return nil, err
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		return nil, err
	}
	defer func() {
		if tempPath != "" {
			_ = ops.Remove(tempPath)
		}
	}()
	if err := os.WriteFile(tempPath, output, 0o644); err != nil {
		return nil, err
	}

	reparsed, err := parser.Parse(output)
	if err != nil {
		return nil, err
	}
	tempGraph, err := graph.Build(reparsed)
	if err != nil {
		return nil, err
	}
	tempCheck := check.Run(tempGraph)
	result.TempCheck = tempCheck.Status
	if tempCheck.Status == core.CheckStatusError {
		result.Code = core.MutationCodeTempCheckError
		result.RecomputeStatus()
		return result, nil
	}

	backupPath, err := replaceWithBackup(opts.InputPath, tempPath, ops)
	if err != nil {
		return nil, err
	}
	result.BackupPath = backupPath
	tempPath = ""

	finalBytes, err := os.ReadFile(opts.InputPath)
	if err != nil {
		return nil, err
	}
	finalParsed, err := parser.Parse(finalBytes)
	if err != nil {
		return nil, err
	}
	finalGraph, err := graph.Build(finalParsed)
	if err != nil {
		return nil, err
	}
	finalCheck := check.Run(finalGraph)
	result.FinalCheck = finalCheck.Status
	if finalCheck.Status == core.CheckStatusError {
		result.Code = core.MutationCodeFinalCheckError
	}
	result.RecomputeStatus()
	return result, nil
}

func replaceWithBackup(inputPath, tempPath string, ops fileOps) (string, error) {
	backupPath, err := nextBackupPath(inputPath)
	if err != nil {
		return "", err
	}
	if err := ops.Rename(inputPath, backupPath); err != nil {
		return "", err
	}
	if err := ops.Rename(tempPath, inputPath); err != nil {
		_ = ops.Rename(backupPath, inputPath)
		return "", fmt.Errorf("replace original: %w", err)
	}
	return backupPath, nil
}

func nextBackupPath(path string) (string, error) {
	candidate := path + ".bak"
	if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate, nil
	}
	for i := 1; i < 1000; i++ {
		candidate = fmt.Sprintf("%s.bak.%d", path, i)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no backup path available for %s", path)
}
