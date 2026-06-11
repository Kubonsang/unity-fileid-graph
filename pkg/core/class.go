package core

func TypeNameForClassID(classID int) string {
	switch classID {
	case 1:
		return "GameObject"
	case 4:
		return "Transform"
	case 23:
		return "MeshRenderer"
	case 33:
		return "MeshFilter"
	case 54:
		return "Rigidbody"
	case 65:
		return "BoxCollider"
	case 114:
		return "MonoBehaviour"
	default:
		return "UNKNOWN"
	}
}
