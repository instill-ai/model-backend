package resource

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func ExtractFromMetadata(ctx context.Context, key string) ([]string, bool) {
	if data, ok := metadata.FromIncomingContext(ctx); !ok {
		return []string{}, false
	} else {
		return data[strings.ToLower(key)], true
	}
}

func GetOwner(ctx context.Context) (string, error) {
	if metadatas, ok := ExtractFromMetadata(ctx, "owner"); ok {
		if len(metadatas) == 0 {
			return "", status.Error(codes.FailedPrecondition, "owner not found in your request")
		}
		return metadatas[0], nil
	} else {
		return "", status.Error(codes.FailedPrecondition, "Error when extract metadata")
	}
}

func GetOwnerFromHeader(r *http.Request) (string, error) {
	owner := r.Header.Get("owner")
	owner = strings.TrimPrefix(owner, "users/") // use user uid for internal process
	return owner, nil
}

func GetID(name string) (string, error) {
	id := strings.TrimPrefix(name, "models/")
	if !strings.HasPrefix(name, "models/") || id == "" {
		return "", status.Error(codes.InvalidArgument, "Error when extract models resource id")
	}
	return id, nil
}

func GetModelInstanceID(name string) (string, string, error) {
	if match, _ := regexp.MatchString(`^models/.+/instances/.+$`, name); !match {
		return "", "", status.Error(codes.InvalidArgument, "Error when extract models instance resource id")
	}
	subs := strings.Split(name, "/")
	return subs[1], subs[3], nil
}

func GetUserNameByUid(uid string) string {
	// TODO request to mgmt-backend
	return "local-user"
}

func GetDefinitionID(name string) (string, error) {
	id := strings.TrimPrefix(name, "model-definitions/")
	if !strings.HasPrefix(name, "model-definitions/") || id == "" {
		return "", status.Error(codes.InvalidArgument, "Error when extract model-definitions resource id")
	}
	return id, nil
}

// GetPermalinkUID returns the resource UID given a resource permalink
func GetPermalinkUID(permalink string) (string, error) {
	uid := permalink[strings.LastIndex(permalink, "/")+1:]
	if uid == "" {
		return "", status.Errorf(codes.InvalidArgument, "Error when extract resource id from resource permalink `%s`", permalink)
	}
	return uid, nil
}

func GetModelInstanceIDFromOperationID(operationID string) (string, uuid.UUID, uuid.UUID, error) {
	// format operationID: {model owner}.{model UID}.{model instance UID}.{timestamp}.{action: deploy/undeploy}
	ids := strings.Split(operationID, ".")
	if len(ids) != 5 {
		return "", uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "Error when extract model instance id from operation id")
	}

	modelUID, err := uuid.FromString(ids[1])
	if err != nil {
		return "", uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "Error when extract model instance id from operation id")
	}

	modelInstanceUID, err := uuid.FromString(ids[2])
	if err != nil {
		return "", uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "Error when extract model instance id from operation id")
	}
	// owner in format users/{uid}, operation id only contain owner uid
	return fmt.Sprintf("users/%s", ids[0]), modelUID, modelInstanceUID, nil
}

func GetOperationID(name string) (string, error) {
	id := strings.TrimPrefix(name, "operations/")
	if !strings.HasPrefix(name, "operations/") || id == "" {
		return "", status.Error(codes.InvalidArgument, "Error when extract operations resource id")
	}
	return id, nil
}
