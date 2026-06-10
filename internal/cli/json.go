package cli

import (
	"encoding/json"
	"io"

	"github.com/Kubonsang/unity-fileid-graph/internal/core"
)

type jsonSummary struct {
	Blocks      int `json:"blocks"`
	GameObjects int `json:"gameobjects"`
	Components  int `json:"components"`
	Transforms  int `json:"transforms"`
	Warnings    int `json:"warnings"`
	Errors      int `json:"errors"`
}

type jsonIssue struct {
	Severity     string `json:"severity"`
	Code         string `json:"code"`
	FileID       int64  `json:"file_id,omitempty"`
	GameObjectID int64  `json:"game_object_id,omitempty"`
	ComponentID  int64  `json:"component_id,omitempty"`
	TransformID  int64  `json:"transform_id,omitempty"`
	ParentID     int64  `json:"parent_id,omitempty"`
	ChildID      int64  `json:"child_id,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Message      string `json:"message,omitempty"`
}

type checkJSONResponse struct {
	Status    string      `json:"status"`
	Namespace string      `json:"namespace"`
	Command   string      `json:"command"`
	File      string      `json:"file"`
	Summary   jsonSummary `json:"summary"`
	Issues    []jsonIssue `json:"issues"`
}

func writeCheckJSON(stdout io.Writer, namespace string, file string, result *core.CheckResult) int {
	response := checkJSONResponse{
		Status:    result.Status,
		Namespace: namespace,
		Command:   "check",
		File:      file,
		Summary: jsonSummary{
			Blocks:      result.BlockCount,
			GameObjects: result.GameObjectCount,
			Components:  result.ComponentCount,
			Transforms:  result.TransformCount,
			Warnings:    len(result.Warnings),
			Errors:      len(result.Errors),
		},
		Issues: []jsonIssue{},
	}
	for _, finding := range result.Errors {
		response.Issues = append(response.Issues, findingToJSONIssue("ERROR", finding))
	}
	for _, finding := range result.Warnings {
		response.Issues = append(response.Issues, findingToJSONIssue("WARN", finding))
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(response)
	if result.Status == core.CheckStatusError {
		return 1
	}
	return 0
}

func findingToJSONIssue(severity string, finding core.CheckFinding) jsonIssue {
	return jsonIssue{
		Severity:     severity,
		Code:         finding.Code,
		FileID:       finding.FileID,
		GameObjectID: finding.GameObjectID,
		ComponentID:  finding.ComponentID,
		TransformID:  finding.TransformID,
		ParentID:     finding.ParentID,
		ChildID:      finding.ChildID,
		Reason:       finding.Reason,
		Message:      finding.Message,
	}
}
