package acl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	openfga "github.com/openfga/api/proto/openfga/v1"

	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/constant"
)

type ACLClient struct {
	writeClient          openfga.OpenFGAServiceClient
	readClient           openfga.OpenFGAServiceClient
	redisClient          *redis.Client
	authorizationModelID string
	storeID              string
}

type Relation struct {
	UID      uuid.UUID
	Relation string
}

type Mode string

const (
	ReadMode  Mode = "read"
	WriteMode Mode = "write"
)

func NewACLClient(wc openfga.OpenFGAServiceClient, rc openfga.OpenFGAServiceClient, redisClient *redis.Client) ACLClient {
	if rc == nil {
		rc = wc
	}
	storeResp, err := wc.ListStores(context.Background(), &openfga.ListStoresRequest{})
	if err != nil {
		panic(err)
	}
	storeID := storeResp.Stores[0].Id

	modelResp, err := wc.ReadAuthorizationModels(context.Background(), &openfga.ReadAuthorizationModelsRequest{
		StoreId: storeID,
	})
	if err != nil {
		panic(err)
	}
	modelID := modelResp.AuthorizationModels[0].Id

	return ACLClient{
		writeClient:          wc,
		readClient:           rc,
		redisClient:          redisClient,
		authorizationModelID: modelID,
		storeID:              storeID,
	}
}

func InitOpenFGAClient(ctx context.Context, host string, port int) (openfga.OpenFGAServiceClient, *grpc.ClientConn) {
	clientDialOpts := grpc.WithTransportCredentials(insecure.NewCredentials())

	clientConn, err := grpc.Dial(
		fmt.Sprintf("%v:%v", host, port),
		clientDialOpts,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.Config.Server.MaxDataSize*constant.MB),
			grpc.MaxCallSendMsgSize(config.Config.Server.MaxDataSize*constant.MB),
		),
	)
	if err != nil {
		panic(err)
	}

	return openfga.NewOpenFGAServiceClient(clientConn), clientConn
}

func (c *ACLClient) getClient(ctx context.Context, mode Mode) openfga.OpenFGAServiceClient {
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	if mode == WriteMode {
		// To solve the read-after-write inconsistency problem,
		// we will direct the user to read from the primary database for a certain time frame
		// to ensure that the data is synchronized from the primary DB to the replica DB.
		_ = c.redisClient.Set(ctx, fmt.Sprintf("db_pin_user:%s:openfga", userUID), time.Now(), time.Duration(config.Config.OpenFGA.Replica.ReplicationTimeFrame)*time.Second)
	}

	// If the user is pinned, we will use the primary database for querying.
	if !errors.Is(c.redisClient.Get(ctx, fmt.Sprintf("db_pin_user:%s:openfga", userUID)).Err(), redis.Nil) {
		return c.writeClient
	}
	if mode == ReadMode {
		return c.readClient
	}
	return c.writeClient
}

func (c *ACLClient) SetOwner(ctx context.Context, objectType string, objectUID uuid.UUID, ownerType string, ownerUID uuid.UUID) error {
	var err error

	data, err := c.getClient(ctx, ReadMode).Read(ctx, &openfga.ReadRequest{
		StoreId: c.storeID,
		TupleKey: &openfga.ReadRequestTupleKey{
			User:     fmt.Sprintf("%s:%s", ownerType, ownerUID.String()),
			Relation: "owner",
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
	})
	if err != nil {
		return err
	}
	if len(data.Tuples) > 0 {
		return nil
	}

	_, err = c.getClient(ctx, WriteMode).Write(ctx, &openfga.WriteRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: c.authorizationModelID,
		Writes: &openfga.WriteRequestWrites{
			TupleKeys: []*openfga.TupleKey{
				{
					User:     fmt.Sprintf("%s:%s", ownerType, ownerUID.String()),
					Relation: "owner",
					Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *ACLClient) SetModelPermission(ctx context.Context, modelUID uuid.UUID, user, role string, enable bool) error {
	var err error
	_ = c.DeleteModelPermission(ctx, modelUID, user)

	if enable {
		_, err = c.getClient(ctx, WriteMode).Write(ctx, &openfga.WriteRequest{
			StoreId:              c.storeID,
			AuthorizationModelId: c.authorizationModelID,
			Writes: &openfga.WriteRequestWrites{
				TupleKeys: []*openfga.TupleKey{
					{
						User:     user,
						Relation: role,
						Object:   fmt.Sprintf("model:%s", modelUID.String()),
					},
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACLClient) DeleteModelPermission(ctx context.Context, modelUID uuid.UUID, user string) error {

	for _, role := range []string{"admin", "writer", "executor", "reader"} {
		_, _ = c.getClient(ctx, WriteMode).Write(ctx, &openfga.WriteRequest{
			StoreId:              c.storeID,
			AuthorizationModelId: c.authorizationModelID,
			Deletes: &openfga.WriteRequestDeletes{
				TupleKeys: []*openfga.TupleKeyWithoutCondition{
					{
						User:     user,
						Relation: role,
						Object:   fmt.Sprintf("model:%s", modelUID.String()),
					},
				},
			},
		})
	}

	return nil
}

func (c *ACLClient) SetPublicModelPermission(ctx context.Context, modelUID uuid.UUID) error {
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

func (c *ACLClient) DeletePublicModelPermission(ctx context.Context, modelUID uuid.UUID) error {
	for _, t := range []string{"user", "visitor"} {
		err := c.DeleteModelPermission(ctx, modelUID, fmt.Sprintf("%s:*", t))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACLClient) Purge(ctx context.Context, objectType string, objectUID uuid.UUID) error {

	data, err := c.getClient(ctx, ReadMode).Read(ctx, &openfga.ReadRequest{
		StoreId: c.storeID,
		TupleKey: &openfga.ReadRequestTupleKey{
			Object: fmt.Sprintf("%s:%s", objectType, objectUID),
		},
	})
	if err != nil {
		return err
	}
	for _, data := range data.Tuples {
		_, err = c.getClient(ctx, WriteMode).Write(ctx, &openfga.WriteRequest{
			StoreId:              c.storeID,
			AuthorizationModelId: c.authorizationModelID,
			Deletes: &openfga.WriteRequestDeletes{
				TupleKeys: []*openfga.TupleKeyWithoutCondition{
					{
						User:     data.Key.User,
						Relation: data.Key.Relation,
						Object:   data.Key.Object,
					},
				},
			},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ACLClient) CheckPermission(ctx context.Context, objectType string, objectUID uuid.UUID, role string) (bool, error) {

	userType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	userUID := ""
	if userType == "user" {
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	} else {
		userUID = resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
	}

	data, err := c.getClient(ctx, ReadMode).Check(ctx, &openfga.CheckRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: c.authorizationModelID,
		TupleKey: &openfga.CheckRequestTupleKey{
			User:     fmt.Sprintf("%s:%s", userType, userUID),
			Relation: role,
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
	})
	if err != nil {
		return false, err
	}
	if data.Allowed {
		return data.Allowed, nil
	}

	return false, nil
}

func (c *ACLClient) CheckPublicExecutable(ctx context.Context, objectType string, objectUID uuid.UUID) (bool, error) {

	data, err := c.getClient(ctx, ReadMode).Check(ctx, &openfga.CheckRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: c.authorizationModelID,
		TupleKey: &openfga.CheckRequestTupleKey{
			User:     "user:*",
			Relation: "executor",
			Object:   fmt.Sprintf("%s:%s", objectType, objectUID.String()),
		},
	})
	if err != nil {
		return false, err
	}
	return data.Allowed, nil
}

func (c *ACLClient) ListPermissions(ctx context.Context, objectType string, role string, isPublic bool) ([]uuid.UUID, error) {

	userType := resource.GetRequestSingleHeader(ctx, constant.HeaderAuthTypeKey)
	userUIDStr := ""
	if userType == "user" {
		userUIDStr = resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)

	} else {
		userUIDStr = resource.GetRequestSingleHeader(ctx, constant.HeaderVisitorUIDKey)
	}

	if isPublic {
		userUIDStr = "*"
	}

	listObjectsResult, err := c.getClient(ctx, ReadMode).ListObjects(ctx, &openfga.ListObjectsRequest{
		StoreId:              c.storeID,
		AuthorizationModelId: c.authorizationModelID,
		User:                 fmt.Sprintf("%s:%s", userType, userUIDStr),
		Relation:             role,
		Type:                 objectType,
	})
	if err != nil {
		return nil, err
	}

	objectUIDs := []uuid.UUID{}
	for _, object := range listObjectsResult.GetObjects() {
		objectUIDs = append(objectUIDs, uuid.FromStringOrNil(strings.Split(object, ":")[1]))
	}

	return objectUIDs, nil
}
