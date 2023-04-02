// Package datamodel contains the data models for the API.
package datamodel

// Error represents the error response.
type Error struct {

	// Error code
	Status int32 `json:"status"`

	// Short description of the error
	Title string `json:"title"`

	// Human-readable error message
	Detail string `json:"detail"`
}

// New creates a new Error.
func (Error) New(status int32, title, detail string) *Error {
	return &Error{Status: status, Title: title, Detail: detail}
}
