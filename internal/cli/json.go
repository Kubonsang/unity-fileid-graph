package cli

import (
	"encoding/json"
	"io"

	"github.com/Kubonsang/unity-fileid-graph/pkg/core"
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

type refsJSONSummary struct {
	References int `json:"references"`
	Warnings   int `json:"warnings"`
}

type refsJSONReference struct {
	BlockFileID int64  `json:"block_file_id"`
	ClassID     int    `json:"class_id"`
	Class       string `json:"class"`
	Field       string `json:"field"`
	FileID      int64  `json:"file_id"`
	GUID        string `json:"guid,omitempty"`
	Type        *int   `json:"type,omitempty"`
}

type refsJSONResponse struct {
	Status     string              `json:"status"`
	Namespace  string              `json:"namespace"`
	Command    string              `json:"command"`
	File       string              `json:"file"`
	Summary    refsJSONSummary     `json:"summary"`
	References []refsJSONReference `json:"references"`
	Issues     []jsonIssue         `json:"issues"`
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

func writeRefsJSON(stdout io.Writer, result *core.RefsResult) int {
	response := refsJSONResponse{
		Status:    result.Status,
		Namespace: result.Namespace,
		Command:   "refs",
		File:      result.File,
		Summary: refsJSONSummary{
			References: len(result.References),
			Warnings:   len(result.Issues),
		},
		References: []refsJSONReference{},
		Issues:     []jsonIssue{},
	}
	for _, reference := range result.References {
		item := refsJSONReference{
			BlockFileID: reference.BlockFileID,
			ClassID:     reference.ClassID,
			Class:       reference.TypeName,
			Field:       reference.Field,
			FileID:      reference.FileID,
			GUID:        reference.GUID,
		}
		if reference.HasType {
			typeValue := reference.Type
			item.Type = &typeValue
		}
		response.References = append(response.References, item)
	}
	for _, issue := range result.Issues {
		response.Issues = append(response.Issues, jsonIssue{
			Severity: "WARN",
			Code:     issue.Code,
			FileID:   issue.FileID,
			Message:  issue.Message,
		})
	}

	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(response)
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
