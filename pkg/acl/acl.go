package acl

import (
	"context"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	openfga "github.com/openfga/go-sdk"
	openfgaClient "github.com/openfga/go-sdk/client"
)

type ACLClient struct {
	client               *openfgaClient.OpenFgaClient
	authorizationModelID *string
}

type Relation struct {
	UID      uuid.UUID
	Relation string
}

func NewACLClient(c *openfgaClient.OpenFgaClient, a *string) ACLClient {
	return ACLClient{
		client:               c,
		authorizationModelID: a,
	}
}

func (c *ACLClient) SetOwner(objectType string, objectUID uuid.UUID, ownerType string, ownerUID uuid.UUID) error {
	var err error
	readOptions := openfgaClient.ClientReadOptions{}
	writeOptions := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	readBody := openfgaClient.ClientReadRequest{
		User:     openfga.PtrString(fmt.Sprintf("%s:%s", ownerType, ownerUID.String())),
		Relation: openfga.PtrString("owner"),
		Object:   openfga.PtrString(fmt.Sprintf("%s:%s", objectType, objectUID.String())),
	}
	data, err := c.client.Read(context.Background()).Body(readBody).Options(readOptions).Execute()
	if err != nil {
		return err
	}
	if len(data.Tuples) > 0 {
		return nil
	}

	writeBody := openfgaClient.ClientWriteRequest{
		Writes: []openfgaClient.ClientTupleKey{
			{
				User:     fmt.Sprintf("%s:%s", ownerType, ownerUID.String()),
				Relation: "owner",
				Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
			}},
	}

	_, err = c.client.Write(context.Background()).Body(writeBody).Options(writeOptions).Execute()
	if err != nil {
		return err
	}
	return nil
}

func (c *ACLClient) SetPublicModelPermission(modelUID uuid.UUID) error {
	// TODO: support fine grained control soon
	for _, t := range []string{"user", "visitor"} {
		err := c.SetModelPermission(modelUID, fmt.Sprintf("%s:*", t), "reader", true)
		if err != nil {
			return err
		}
	}
	err := c.SetModelPermission(modelUID, "user:*", "executor", true)
	if err != nil {
		return err
	}

	return nil
}

func (c *ACLClient) DeletePublicModelPermission(modelUID uuid.UUID) error {
	for _, t := range []string{"user", "visitor"} {
		err := c.DeleteModelPermission(modelUID, fmt.Sprintf("%s:*", t))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACLClient) SetModelPermission(modelUID uuid.UUID, user, role string, enable bool) error {
	var err error
	options := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	_ = c.DeleteModelPermission(modelUID, user)

	if enable {
		body := openfgaClient.ClientWriteRequest{
			Writes: []openfgaClient.ClientContextualTupleKey{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("model_:%s", modelUID.String()),
				}},
		}

		_, err = c.client.Write(context.Background()).Body(body).Options(options).Execute()
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACLClient) DeleteModelPermission(modelUID uuid.UUID, user string) error {
	// var err error
	options := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	for _, role := range []string{"admin", "writer", "executor", "reader"} {
		body := openfgaClient.ClientWriteRequest{
			Deletes: []openfgaClient.ClientTupleKeyWithoutCondition{
				{
					User:     user,
					Relation: role,
					Object:   fmt.Sprintf("model_:%s", modelUID.String()),
				}}}
		_, _ = c.client.Write(context.Background()).Body(body).Options(options).Execute()
	}

	return nil
}

func (c *ACLClient) Purge(objectType string, objectUID uuid.UUID) error {
	readOptions := openfgaClient.ClientReadOptions{}
	writeOptions := openfgaClient.ClientWriteOptions{
		AuthorizationModelId: c.authorizationModelID,
	}

	readBody := openfgaClient.ClientReadRequest{
		Object: openfga.PtrString(fmt.Sprintf("%s:%s", objectType, objectUID)),
	}
	resp, err := c.client.Read(context.Background()).Body(readBody).Options(readOptions).Execute()
	if err != nil {
		return err
	}
	for _, data := range resp.Tuples {
		body := openfgaClient.ClientWriteRequest{
			Deletes: []openfgaClient.ClientTupleKeyWithoutCondition{
				{
					User:     data.Key.User,
					Relation: data.Key.Relation,
					Object:   data.Key.Object,
				}}}
		_, err := c.client.Write(context.Background()).Body(body).Options(writeOptions).Execute()

		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACLClient) CheckPermission(objectType string, objectUID uuid.UUID, userType string, userUID uuid.UUID, role string) (bool, error) {
	options := openfgaClient.ClientCheckOptions{
		AuthorizationModelId: c.authorizationModelID,
	}
	body := openfgaClient.ClientCheckRequest{
		User:     fmt.Sprintf("%s:%s", userType, userUID.String()),
		Relation: role,
		Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
	}
	data, err := c.client.Check(context.Background()).Body(body).Options(options).Execute()
	if err != nil {
		return false, err
	}
	if *data.Allowed {
		return *data.Allowed, nil
	}

	return false, nil
}

func (c *ACLClient) CheckPublicExecutable(objectType string, objectUID uuid.UUID) (bool, error) {

	options := openfgaClient.ClientCheckOptions{
		AuthorizationModelId: c.authorizationModelID,
	}
	body := openfgaClient.ClientCheckRequest{
		User:     "user:*",
		Relation: "executor",
		Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
	}
	data, err := c.client.Check(context.Background()).Body(body).Options(options).Execute()
	if err != nil {
		return false, err
	}
	if *data.Allowed {
		return *data.Allowed, nil
	}

	return *data.Allowed, nil
}

func (c *ACLClient) ListPermissions(objectType string, userType string, userUID uuid.UUID, role string) ([]uuid.UUID, error) {

	options := openfgaClient.ClientListObjectsOptions{
		AuthorizationModelId: c.authorizationModelID,
	}
	userUIDStr := "*"
	if userUID != uuid.Nil {
		userUIDStr = userUID.String()
	}

	body := openfgaClient.ClientListObjectsRequest{
		User:     fmt.Sprintf("%s:%s", userType, userUIDStr),
		Relation: role,
		Type:     objectType,
	}
	listObjectsResult, err := c.client.ListObjects(context.Background()).Body(body).Options(options).Execute()
	if err != nil {
		return nil, err
	}
	objectUIDs := []uuid.UUID{}
	for _, object := range listObjectsResult.GetObjects() {
		objectUIDs = append(objectUIDs, uuid.FromStringOrNil(strings.Split(object, ":")[1]))
	}

	return objectUIDs, nil
}
