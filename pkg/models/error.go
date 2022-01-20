/*
 * Model Server
 *
 * This is API specification of model server
 *
 * API version: 0.0.1
 * Contact: hello@instill.tech
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package models

type Error struct {

	// Error code
	Status int32 `json:"status"`

	// Short description of the error
	Title string `json:"title"`

	// Human-readable error message
	Detail string `json:"detail"`

	// The duration in milliseconds (s) it takes for a request to be processed
	Duration float64 `json:"duration"`
}
