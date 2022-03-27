package datamodel

type Error struct {

	// Error code
	Status int32 `json:"status"`

	// Short description of the error
	Title string `json:"title"`

	// Human-readable error message
	Detail string `json:"detail"`
}
