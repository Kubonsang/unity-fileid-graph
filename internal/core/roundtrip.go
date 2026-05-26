package core

const (
	RoundtripModeLosslessBlockCopy = "lossless-block-copy"

	RoundtripStatusOK    = "OK"
	RoundtripStatusError = "ERROR"

	EditorOpenNotChecked = "NOT_CHECKED"
)

type RoundtripResult struct {
	Status             string
	Mode               string
	OutputPath         string
	BytesEqual         bool
	Reparsed           bool
	BlockSequenceEqual bool
	GraphCheckStatus   string
	LineEndingStyle    string
	EditorOpenStatus   string
}

func (r *RoundtripResult) RecomputeStatus() {
	if r.BytesEqual && r.Reparsed && r.BlockSequenceEqual && r.GraphCheckStatus == CheckStatusOK {
		r.Status = RoundtripStatusOK
		return
	}

	r.Status = RoundtripStatusError
}
