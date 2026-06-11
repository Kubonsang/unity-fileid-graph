package core

const (
	CheckStatusOK    = "OK"
	CheckStatusWarn  = "WARN"
	CheckStatusError = "ERROR"
)

const (
	CheckDuplicateFileID               = "DUPLICATE_FILE_ID"
	CheckMissingComponentBlock         = "MISSING_COMPONENT_BLOCK"
	CheckMissingGameObjectBlock        = "MISSING_GAMEOBJECT_BLOCK"
	CheckGoComponentBackrefMismatch    = "GO_COMPONENT_BACKREF_MISMATCH"
	CheckTransformParentChildMismatch  = "TRANSFORM_PARENT_CHILD_MISMATCH"
	CheckMissingTransformComponent     = "MISSING_TRANSFORM_COMPONENT"
	CheckSuspiciousMonoBehaviourScript = "SUSPICIOUS_MONOBEHAVIOUR_SCRIPT"
)

type CheckFinding struct {
	Code           string
	DuplicateCount int
	FileID         int64
	GameObjectID   int64
	ComponentID    int64
	TransformID    int64
	ParentID       int64
	ChildID        int64
	Reason         string
	Message        string
}

type CheckResult struct {
	Status          string
	BlockCount      int
	GameObjectCount int
	ComponentCount  int
	TransformCount  int
	Errors          []CheckFinding
	Warnings        []CheckFinding
}

func (result *CheckResult) RecomputeStatus() {
	switch {
	case len(result.Errors) > 0:
		result.Status = CheckStatusError
	case len(result.Warnings) > 0:
		result.Status = CheckStatusWarn
	default:
		result.Status = CheckStatusOK
	}
}
