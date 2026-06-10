package core

import "testing"

func TestTypeNameForClassIDMapsKnownClasses(t *testing.T) {
	tests := []struct {
		classID int
		want    string
	}{
		{classID: 1, want: "GameObject"},
		{classID: 4, want: "Transform"},
		{classID: 23, want: "MeshRenderer"},
		{classID: 33, want: "MeshFilter"},
		{classID: 54, want: "Rigidbody"},
		{classID: 65, want: "BoxCollider"},
		{classID: 114, want: "MonoBehaviour"},
		{classID: 999999, want: "UNKNOWN"},
	}

	for _, tc := range tests {
		if got := TypeNameForClassID(tc.classID); got != tc.want {
			t.Fatalf("classID %d: expected %q, got %q", tc.classID, tc.want, got)
		}
	}
}
