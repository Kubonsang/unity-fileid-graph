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
	CheckTransformParentCycle          = "TRANSFORM_PARENT_CYCLE"
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

	// Transform parent/child links the symmetry check could not assert and
	// deliberately skipped, so a passing check is never a silent "skipped
	// everything". SkippedLinks is the total; the breakdown explains why:
	//   stripped        — endpoint is a stripped nested prefab-instance block
	//                      (no locally authoritative m_Father/m_Children)
	//   unmodeled class — endpoint block exists but its class is not modeled as
	//                      a transform (e.g. RectTransform 224)
	SkippedLinks          int
	SkippedStripped       int
	SkippedUnmodeledClass int
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
