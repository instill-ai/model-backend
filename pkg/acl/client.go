package acl

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	openfgaclient "github.com/openfga/go-sdk/client"

	openfgax "github.com/instill-ai/x/openfga"
)

// Model-specific object types
const (
	ObjectTypeModel        openfgax.ObjectType = "model_"
	ObjectTypeOrganization openfgax.ObjectType = "organization"
)

// aclClient wraps the x/openfga Client with model-backend specific operations
type aclClient struct {
	openfgax.Client
}

// ACLClientInterface defines the interface for model-backend ACL operations
type ACLClientInterface interface {
	openfgax.Client

	CheckPublicExecutable(ctx context.Context, objectType openfgax.ObjectType, objectUID uuid.UUID) (bool, error)
	SetModelPermission(ctx context.Context, modelUID uuid.UUID, user string, role string, enable bool) error
	SetPublicModelPermission(ctx context.Context, modelUID uuid.UUID) error
	DeleteModelPermission(ctx context.Context, modelUID uuid.UUID, user string) error
	DeletePublicModelPermission(ctx context.Context, modelUID uuid.UUID) error
}

// NewFGAClient creates a new model-backend specific FGA client
func NewFGAClient(client openfgax.Client) ACLClientInterface {
	return &aclClient{Client: client}
}

// CheckPublicExecutable checks if public users can execute an object
func (c *aclClient) CheckPublicExecutable(ctx context.Context, objectType openfgax.ObjectType, objectUID uuid.UUID) (bool, error) {
	body := openfgaclient.ClientCheckRequest{
		User:     fmt.Sprintf("%s:*", openfgax.OwnerTypeUser),
		Relation: "executor",
		Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
	}
	data, err := c.SDKClient().Check(ctx).Body(body).Execute()
	if err != nil {
		return false, err
	}
	return *data.Allowed, nil
}

// SetModelPermission sets a specific permission for a user on a model
func (c *aclClient) SetModelPermission(ctx context.Context, modelUID uuid.UUID, user string, role string, enable bool) error {
	// First delete existing permission for this user
	_ = c.DeleteModelPermission(ctx, modelUID, user)

	if enable {
		writeBody := openfgaclient.ClientWriteRequest{
			Writes: []openfgaclient.ClientTupleKey{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("%s:%s", ObjectTypeModel, modelUID.String()),
				},
			},
		}
		_, err := c.SDKClient().Write(ctx).Body(writeBody).Execute()
		return err
	}

	return nil
}

// SetPublicModelPermission sets public permissions for a model
func (c *aclClient) SetPublicModelPermission(ctx context.Context, modelUID uuid.UUID) error {
	for _, t := range []string{"user", "visitor"} {
		err := c.SetModelPermission(ctx, modelUID, fmt.Sprintf("%s:*", t), "reader", true)
		if err != nil {
			return err
		}
	}
	err := c.SetModelPermission(ctx, modelUID, "user:*", "executor", true)
	if err != nil {
		return err
	}

	return nil
}

// DeleteModelPermission deletes all permissions for a user on a model
func (c *aclClient) DeleteModelPermission(ctx context.Context, modelUID uuid.UUID, user string) error {
	for _, role := range []string{"admin", "writer", "executor", "reader"} {
		deleteBody := openfgaclient.ClientWriteRequest{
			Deletes: []openfgaclient.ClientTupleKeyWithoutCondition{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("%s:%s", ObjectTypeModel, modelUID.String()),
				},
			},
		}
		_, _ = c.SDKClient().Write(ctx).Body(deleteBody).Execute()
	}

	return nil
}

// DeletePublicModelPermission deletes public permissions for a model
func (c *aclClient) DeletePublicModelPermission(ctx context.Context, modelUID uuid.UUID) error {
	for _, t := range []string{"user", "visitor"} {
		err := c.DeleteModelPermission(ctx, modelUID, fmt.Sprintf("%s:*", t))
		if err != nil {
			return err
		}
	}

	return nil
}
