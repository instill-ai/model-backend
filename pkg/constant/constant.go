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
const HeaderVisitorUIDKey = "Instill-Visitor-Uid"
const HeaderAuthTypeKey = "Instill-Auth-Type"
const HeaderRequesterUIDKey = "Instill-Requester-Uid"
const HeaderUserAgent = "Instill-User-Agent"

// Ray proto path
const RayProtoPath string = "assets/ray/proto"

const ContentTypeJSON = "application/json"
