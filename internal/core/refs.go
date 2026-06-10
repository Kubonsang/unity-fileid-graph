package core

const (
	RefsStatusOK   = "OK"
	RefsStatusWarn = "WARN"
)

type Reference struct {
	BlockFileID int64
	ClassID     int
	TypeName    string
	Field       string
	FileID      int64
	GUID        string
	Type        int
	HasGUID     bool
	HasType     bool
	RawValue    string
}

type RefsResult struct {
	Status     string
	Namespace  string
	File       string
	References []Reference
	Issues     []Issue
}

func (result *RefsResult) RecomputeStatus() {
	if len(result.Issues) > 0 {
		result.Status = RefsStatusWarn
		return
	}
	result.Status = RefsStatusOK
}
