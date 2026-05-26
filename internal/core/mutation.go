package core

const (
	MutationStatusOK      = "OK"
	MutationStatusWarn    = "WARN"
	MutationStatusError   = "ERROR"
	MutationStatusBlocked = "BLOCKED"
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

func (r *SetResult) MarkBlocked(message string) {
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
