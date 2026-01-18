package resource

import (
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type NamespaceType string

const (
	User         NamespaceType = "users"
	Organization NamespaceType = "organizations"
)

type Namespace struct {
	NsType NamespaceType
	NsID   string
	NsUID  uuid.UUID
}

func (ns Namespace) Name() string {
	return fmt.Sprintf("%s/%s", ns.NsType, ns.NsID)
}
func (ns Namespace) Permalink() string {
	return fmt.Sprintf("%s/%s", ns.NsType, ns.NsUID.String())
}

func GetDefinitionID(name string) (string, error) {
	id := strings.TrimPrefix(name, "model-definitions/")
	if !strings.HasPrefix(name, "model-definitions/") || id == "" {
		return "", status.Error(codes.InvalidArgument, "Error when extract model-definitions resource id")
	}
	return id, nil
}

// GetRscPermalinkUID returns the resource UID given a resource permalink
func GetRscPermalinkUID(path string) (uuid.UUID, error) {
	splits := strings.Split(path, "/")
	if len(splits) < 2 {
		return uuid.Nil, fmt.Errorf("error when extract resource id from resource permalink '%s'", path)
	}

	return uuid.FromStringOrNil(splits[1]), nil
}

// GetNamespaceTypeAndUID returns the namespace type and uid from permalink
func GetNamespaceTypeAndUID(permalink string) (string, uuid.UUID, error) {
	splits := strings.Split(permalink, "/")
	if len(splits) < 2 {
		return "", uuid.Nil, fmt.Errorf("error when extract resource id from resource permalink '%s'", permalink)
	}

	return splits[0], uuid.FromStringOrNil(splits[1]), nil
}

func GetWorkflowID(operationID string) (string, error) {
	id := strings.TrimPrefix(operationID, "operations/")
	if !strings.HasPrefix(operationID, "operations/") || id == "" {
		return "", status.Error(codes.InvalidArgument, "Error when extract operations resource id")
	}
	return id, nil
}
