package core

const (
	MutationStatusOK           = "OK"
	MutationStatusWarn         = "WARN"
	MutationStatusError        = "ERROR"
	MutationStatusBlocked      = "BLOCKED"
	MutationStatusExperimental = "EXPERIMENTAL"
)

const (
	MutationCodeMonoBehaviourWriteBlocked = "MONOBEHAVIOUR_NATIVE_WRITE_BLOCKED"
	MutationCodeStrippedObjectBlocked     = "STRIPPED_OBJECT_MUTATION_BLOCKED"
	MutationCodeUnsupportedFieldShape     = "UNSUPPORTED_FIELD_SHAPE"
	MutationCodeFieldNotFound             = "FIELD_NOT_FOUND"
	MutationCodeFileIDNotFound            = "FILE_ID_NOT_FOUND"
	MutationCodeDuplicateFileID           = "DUPLICATE_FILE_ID"
	MutationCodePrecheckError             = "PRECHECK_ERROR"
	MutationCodeTempCheckError            = "TEMP_CHECK_ERROR"
	MutationCodeFinalCheckError           = "FINAL_CHECK_ERROR"
	MutationCodeTransformRemoveBlocked    = "TRANSFORM_REMOVE_BLOCKED"
	MutationCodeUnsupportedComponentClass = "UNSUPPORTED_COMPONENT_CLASS"
	MutationCodeComponentOwnerNotFound    = "COMPONENT_OWNER_NOT_FOUND"
	MutationCodeComponentOwnerMismatch    = "COMPONENT_OWNER_MISMATCH"
	MutationCodeComponentRefNotFound      = "COMPONENT_REF_NOT_FOUND"
	MutationCodeUnsupportedComponentList  = "UNSUPPORTED_COMPONENT_LIST_SHAPE"
	MutationCodeDanglingFileID            = "DANGLING_FILE_ID"
	MutationCodeExperimentalFlagRequired  = "EXPERIMENTAL_FLAG_REQUIRED"
	MutationCodeWriteFlagRequired         = "WRITE_FLAG_REQUIRED"
)

type SetOptions struct {
	InputPath string
	FileID    int64
	Field     string
	Value     string
}

type SetResult struct {
	Status     string
	Code       string
	FileID     int64
	Field      string
	OldValue   string
	NewValue   string
	PreCheck   string
	TempCheck  string
	FinalCheck string
	BackupPath string
	Message    string
}

type RemoveComponentOptions struct {
	InputPath    string
	FileID       int64
	Experimental bool
	Write        bool
}

type RemoveComponentResult struct {
	Status     string
	Code       string
	FileID     int64
	ClassID    int
	TypeName   string
	GameObject int64
	PreCheck   string
	TempCheck  string
	FinalCheck string
	BackupPath string
	Message    string
}

func (r *SetResult) MarkBlocked(message string) {
	r.Status = MutationStatusBlocked
	r.Message = message
}

func (r *RemoveComponentResult) MarkBlocked(message string) {
	r.Status = MutationStatusBlocked
	r.Message = message
}

func (r *SetResult) RecomputeStatus() {
	if r.Status == MutationStatusBlocked {
		return
	}
	if r.PreCheck == CheckStatusError || r.TempCheck == CheckStatusError || r.FinalCheck == CheckStatusError {
		r.Status = MutationStatusError
		return
	}
	if r.PreCheck == CheckStatusWarn || r.TempCheck == CheckStatusWarn || r.FinalCheck == CheckStatusWarn {
		r.Status = MutationStatusWarn
		return
	}
	r.Status = MutationStatusOK
}

func (r *RemoveComponentResult) RecomputeStatus() {
	if r.Status == MutationStatusBlocked {
		return
	}
	if r.PreCheck == CheckStatusError || r.TempCheck == CheckStatusError || r.FinalCheck == CheckStatusError {
		r.Status = MutationStatusError
		return
	}
	r.Status = MutationStatusExperimental
}
