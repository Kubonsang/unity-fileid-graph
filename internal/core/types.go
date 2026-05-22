package core

type Block struct {
	Index      int
	ClassID    int
	FileID     int64
	HeaderRaw  string
	BodyRaw    string
	IsStripped bool
	StartLine  int
	EndLine    int
}

type ParseResult struct {
	Blocks      []*Block
	PreambleRaw string
	TrailerRaw  string
}
