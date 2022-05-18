package main

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func appendCustomHeaderMiddleware(next runtime.HandlerFunc) runtime.HandlerFunc {
	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		// TODO: Replace with decoded JWT header
		r.Header.Add("owner", "users/2a06c2f7-8da9-4046-91ea-240f88a5d729")
		next(w, r, pathParams)
	})
}
