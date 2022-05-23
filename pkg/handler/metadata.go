package handler

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func extractFromMetadata(ctx context.Context, key string) ([]string, bool) {
	if data, ok := metadata.FromIncomingContext(ctx); !ok {
		return []string{}, false
	} else {
		return data[strings.ToLower(key)], true
	}
}

func getOwner(ctx context.Context) (string, error) {
	if metadatas, ok := extractFromMetadata(ctx, "owner"); ok {
		if len(metadatas) == 0 {
			return "", status.Error(codes.FailedPrecondition, "owner not found in your request")
		}
		//TODO: get user info from mgmt-backend
		return "users/local-user", nil
	} else {
		return "", status.Error(codes.FailedPrecondition, "Error when extract metadata")
	}
}

func getOwnerFromHeader(r *http.Request) (string, error) {
	owner := r.Header.Get("owner")
	fmt.Println("owner", owner)
	//TODO: get user info from mgmt-backend
	return "users/local-user", nil
}

func getID(name string) (string, error) {
	id := strings.TrimPrefix(name, "models/")
	if !strings.HasPrefix(name, "models/") || id == "" {
		return "", status.Error(codes.InvalidArgument, "Error when extract models resource id")
	}
	return id, nil
}

func getModelInstanceID(name string) (string, string, error) {
	if match, _ := regexp.MatchString(`^models/.+/instances/.+$`, name); !match {
		return "", "", status.Error(codes.InvalidArgument, "Error when extract models instance resource id")
	}
	subs := strings.Split(name, "/")
	return subs[1], subs[3], nil
}

func getDefinitionUID(name string) (string, error) {
	id := strings.TrimPrefix(name, "model-definitions/")
	if !strings.HasPrefix(name, "model-definitions/") || id == "" {
		return "", status.Error(codes.InvalidArgument, "Error when extract model-definitions resource id")
	}
	return id, nil
}
