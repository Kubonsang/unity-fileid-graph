package core

const (
	IssueTabIndent               = "TAB_INDENT"
	IssueUnknownFieldShape       = "UNKNOWN_FIELD_SHAPE"
	IssueUnknownClassID          = "UNKNOWN_CLASS_ID"
	IssueUnsupportedComponentRef = "UNSUPPORTED_COMPONENT_REF"
)

type UnityObject struct {
	FileID   int64
	ClassID  int
	TypeName string
	Block    *Block
}

// ScriptRef is an external PPtr and not a local block fileID in this file.
type ScriptRef struct {
	FileID int64
	GUID   string
	Type   int
}

type GameObjectNode struct {
	FileID     int64
	Name       string
	Components []int64
	Transform  int64
}

type ComponentNode struct {
	FileID        int64
	ClassID       int
	TypeName      string
	GameObject    int64
	HasGameObject bool
	Script        *ScriptRef
}

type TransformNode struct {
	FileID     int64
	GameObject int64
	Father     int64
	Children   []int64
}

type Issue struct {
	Code    string
	FileID  int64
	Message string
}

type Graph struct {
	Blocks      []*Block
	BlocksByID  map[int64][]*Block
	ObjectsByID map[int64][]*UnityObject
	GameObjects map[int64]*GameObjectNode
	Components  map[int64]*ComponentNode
	Transforms  map[int64]*TransformNode
	Issues      []Issue
}
