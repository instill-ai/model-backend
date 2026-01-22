package acl

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	openfga "github.com/openfga/api/proto/openfga/v1"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/x/client"

	aclx "github.com/instill-ai/x/acl"
)

// ACLClientInterface is an interface for the ACL client.
type ACLClientInterface interface {
	SetOwner(ctx context.Context, objectType string, objectUID uuid.UUID, ownerType string, ownerUID uuid.UUID) error
	SetModelPermission(ctx context.Context, modelUID uuid.UUID, user, role string, enable bool) error
	SetPublicModelPermission(ctx context.Context, modelUID uuid.UUID) error
	DeleteModelPermission(ctx context.Context, modelUID uuid.UUID, user string) error
	DeletePublicModelPermission(ctx context.Context, modelUID uuid.UUID) error
	Purge(ctx context.Context, objectType string, objectUID uuid.UUID) error
	CheckPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error)
	CheckPublicExecutable(ctx context.Context, objectType string, objectUID uuid.UUID) (bool, error)
	ListPermissions(ctx context.Context, objectType string, role string, isPublic bool) ([]uuid.UUID, error)
}

// ACLClient wraps the shared ACL client and adds model-specific methods.
type ACLClient struct {
	*aclx.ACLClient
}

// Relation is a relation for the ACL.
type Relation struct {
	UID      uuid.UUID
	Relation string
}

type Mode string
type ObjectType string
type Role string

const (
	ReadMode  Mode = "read"
	WriteMode Mode = "write"

	Organization ObjectType = "organization"

	Member Role = "member"

	// ModelObject is the OpenFGA object type for models (note the underscore suffix)
	ModelObject = "model_"
)

// NewACLClient creates a new ACL client using the shared library.
func NewACLClient(wc openfga.OpenFGAServiceClient, rc openfga.OpenFGAServiceClient, redisClient *redis.Client) *ACLClient {
	cfg := aclx.Config{
		Host: config.Config.OpenFGA.Host,
		Port: config.Config.OpenFGA.Port,
		Replica: aclx.ReplicaConfig{
			Host:                 config.Config.OpenFGA.Replica.Host,
			Port:                 config.Config.OpenFGA.Replica.Port,
			ReplicationTimeFrame: config.Config.OpenFGA.Replica.ReplicationTimeFrame,
		},
		Cache: aclx.CacheConfig{
			Enabled: config.Config.OpenFGA.Cache.Enabled,
			TTL:     config.Config.OpenFGA.Cache.TTL,
		},
	}

	sharedClient := aclx.NewClient(wc, rc, redisClient, cfg)

	return &ACLClient{
		ACLClient: sharedClient,
	}
}

// InitOpenFGAClient initializes gRPC connections to OpenFGA server.
func InitOpenFGAClient(ctx context.Context, host string, port int) (openfga.OpenFGAServiceClient, *grpc.ClientConn) {
	return aclx.InitOpenFGAClient(ctx, host, port, client.MaxPayloadSize/(1024*1024))
}

// SetModelPermission sets a permission for a user on a model.
func (c *ACLClient) SetModelPermission(ctx context.Context, modelUID uuid.UUID, user, role string, enable bool) error {
	return c.SetResourcePermission(ctx, ModelObject, modelUID, user, role, enable)
}

// DeleteModelPermission deletes all permissions for a user on a model.
func (c *ACLClient) DeleteModelPermission(ctx context.Context, modelUID uuid.UUID, user string) error {
	return c.DeleteResourcePermission(ctx, ModelObject, modelUID, user)
}

// SetPublicModelPermission sets public permissions on a model.
func (c *ACLClient) SetPublicModelPermission(ctx context.Context, modelUID uuid.UUID) error {
	return c.SetPublicPermission(ctx, ModelObject, modelUID)
}

// DeletePublicModelPermission deletes public permissions from a model.
func (c *ACLClient) DeletePublicModelPermission(ctx context.Context, modelUID uuid.UUID) error {
	return c.DeletePublicPermission(ctx, ModelObject, modelUID)
}

// ListPermissions lists all objects of a type that the current user has a role for.
func (c *ACLClient) ListPermissions(ctx context.Context, objectType string, role string, isPublic bool) ([]uuid.UUID, error) {
	return c.ACLClient.ListPermissions(ctx, objectType, role, isPublic)
}
