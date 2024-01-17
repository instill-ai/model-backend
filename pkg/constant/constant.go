package constant

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

// Constants for resource owner
const DefaultUserID string = "admin"
const HeaderUserUIDKey = "Instill-User-Uid"

// Ray proto path
const RayProtoPath string = "assets/ray/proto"
