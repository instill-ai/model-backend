package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/instill-ai/model-backend/pkg/repository"
	"github.com/instill-ai/model-backend/pkg/service"
)

type fn func(service.Service, repository.Repository, http.ResponseWriter, *http.Request, map[string]string)

// AppendCustomHeaderMiddleware appends custom header to the response.
func AppendCustomHeaderMiddleware(s service.Service, repo repository.Repository, next fn) runtime.HandlerFunc {
	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		next(s, repo, w, r, pathParams)
	})
}

// HandleProfileImage handles the profile image request.
func HandleProfileImage(s service.Service, r repository.Repository, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
	defer cancel()
	if v, ok := pathParams["path"]; !ok || len(strings.Split(v, "/")) < 4 {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	namespaceID := strings.Split(pathParams["path"], "/")[1]
	modelID := strings.Split(pathParams["path"], "/")[3]

	ns, err := s.GetRscNamespace(ctx, namespaceID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	profileImageBase64 := ""
	dbModel, err := r.GetNamespaceModelByID(ctx, ns.Permalink(), modelID, true, true)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if dbModel.ProfileImage.String == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	profileImageBase64 = dbModel.ProfileImage.String

	b, err := base64.StdEncoding.DecodeString(strings.Split(profileImageBase64, ",")[1])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/octet-stream")
	_, err = w.Write(b)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
}
