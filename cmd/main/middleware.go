package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/instill-ai/model-backend/internal/external"
	"github.com/instill-ai/model-backend/pkg/repository"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
)

func appendCustomHeaderMiddleware(next runtime.HandlerFunc) runtime.HandlerFunc {
	return runtime.HandlerFunc(func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		// TODO: Replace with decoded JWT header
		userServiceClient, userServiceClientConn := external.InitMgmtAdminServiceClient()
		defer userServiceClientConn.Close()
		userPageToken := ""
		userPageSizeMax := int64(repository.MaxPageSize)
		userResp, err := userServiceClient.ListUser(context.Background(), &mgmtPB.ListUserRequest{
			PageSize:  &userPageSizeMax,
			PageToken: &userPageToken,
		})
		if err == nil && len(userResp.Users) > 0 && userResp.Users[0].GetUid() != "" {
			r.Header.Add("owner", fmt.Sprintf("users/%s", userResp.Users[0].GetUid()))
		} else {
			r.Header.Add("owner", "users/45d19b6d-5073-4bc7-b3c6-b668ea98b3c4")
		}
		next(w, r, pathParams)
	})
}
