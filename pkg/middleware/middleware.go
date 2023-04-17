package middleware

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/instill-ai/model-backend/pkg/constant"
)

func AppendCustomHeaderMiddleware(next runtime.HandlerFunc) runtime.HandlerFunc {
	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		r.Header.Add(constant.HeaderOwnerIDKey, constant.DefaultOwnerID)
		next(w, r, pathParams)
	})
}
