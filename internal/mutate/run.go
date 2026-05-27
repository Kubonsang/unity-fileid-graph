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

type writePipelineResult struct {
	TempCheck   string
	FinalCheck  string
	BackupPath  string
	Restored    bool
	RestoreFail bool
}

type writePipelineCheckPhase string

const (
	writePipelineCheckTemp  writePipelineCheckPhase = "temp"
	writePipelineCheckFinal writePipelineCheckPhase = "final"
)

type writePipelineOptions struct {
	RestoreOnFinalCheckError bool
	CheckBytes               func(phase writePipelineCheckPhase, bytes []byte) (string, error)
	AfterReplace             func(path string) error
}

func RunSet(opts core.SetOptions) (*core.SetResult, error) {
	return runSetWithFileOps(opts, defaultFileOps())
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
	pipeline, err := completeWritePipeline(opts.InputPath, output, ops, writePipelineOptions{
		RestoreOnFinalCheckError: false,
	})
	if err != nil {
		return nil, err
	}
	result.TempCheck = pipeline.TempCheck
	result.FinalCheck = pipeline.FinalCheck
	result.BackupPath = pipeline.BackupPath
	if result.TempCheck == core.CheckStatusError {
		result.Code = core.MutationCodeTempCheckError
		result.RecomputeStatus()
		return result, nil
	}
	if result.FinalCheck == core.CheckStatusError {
		result.Code = core.MutationCodeFinalCheckError
	}
	result.RecomputeStatus()
	return result, nil
}

func defaultFileOps() fileOps {
	return fileOps{
		Rename: os.Rename,
		Remove: os.Remove,
	}
}

func completeWritePipeline(inputPath string, output []byte, ops fileOps, options writePipelineOptions) (writePipelineResult, error) {
	result := writePipelineResult{}

	dir := filepath.Dir(inputPath)
	tempFile, err := os.CreateTemp(dir, filepath.Base(inputPath)+".tmp-*")
	if err != nil {
		return result, err
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		return result, err
	}
	defer func() {
		if tempPath != "" {
			_ = ops.Remove(tempPath)
		}
	}()
	if err := os.WriteFile(tempPath, output, 0o644); err != nil {
		return result, err
	}

	tempStatus, err := checkBytesWithOptions(writePipelineCheckTemp, output, options)
	if err != nil {
		return result, err
	}
	result.TempCheck = tempStatus
	if tempStatus == core.CheckStatusError {
		return result, nil
	}

	backupPath, err := replaceWithBackup(inputPath, tempPath, ops)
	if err != nil {
		return result, err
	}
	result.BackupPath = backupPath
	tempPath = ""

	if options.AfterReplace != nil {
		if err := options.AfterReplace(inputPath); err != nil {
			return result, err
		}
	}

	finalBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return result, err
	}
	finalStatus, err := checkBytesWithOptions(writePipelineCheckFinal, finalBytes, options)
	if err != nil {
		return result, err
	}
	result.FinalCheck = finalStatus
	if finalStatus == core.CheckStatusError && options.RestoreOnFinalCheckError {
		if restoreErr := restoreFromBackup(inputPath, backupPath, ops); restoreErr != nil {
			result.RestoreFail = true
		} else {
			result.Restored = true
		}
	}

	return result, nil
}

func checkBytesWithOptions(phase writePipelineCheckPhase, output []byte, options writePipelineOptions) (string, error) {
	if options.CheckBytes != nil {
		return options.CheckBytes(phase, output)
	}

	reparsed, err := parser.Parse(output)
	if err != nil {
		return "", err
	}
	tempGraph, err := graph.Build(reparsed)
	if err != nil {
		return "", err
	}
	return check.Run(tempGraph).Status, nil
}

func restoreFromBackup(inputPath, backupPath string, ops fileOps) error {
	if err := ops.Rename(backupPath, inputPath); err == nil {
		return nil
	}
	_ = ops.Remove(inputPath)
	return ops.Rename(backupPath, inputPath)
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
