package convert000008

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	openfga "github.com/openfga/api/proto/openfga/v1"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/datamodel"
)

const batchSize = 100

// ModelACLConverter executes code along with the 8th database
// schema revision.
type ModelACLConverter struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

// Migrate updates the `TASK_JQ` input in the JSON operator to kebab-case.
func (c *ModelACLConverter) Migrate() error {
	if err := c.migrateModel(); err != nil {
		return err
	}

	return nil
}

func (c *ModelACLConverter) migrateModel() error {
	ctx := context.Background()

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	fgaClient, fgaClientConn := acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Host, config.Config.OpenFGA.Port)
	if fgaClientConn != nil {
		defer fgaClientConn.Close()
	}
	var fgaReplicaClient openfga.OpenFGAServiceClient
	var fgaReplicaClientConn *grpc.ClientConn
	if config.Config.OpenFGA.Replica.Host != "" {

		fgaReplicaClient, fgaReplicaClientConn = acl.InitOpenFGAClient(ctx, config.Config.OpenFGA.Replica.Host, config.Config.OpenFGA.Replica.Port)
		if fgaReplicaClientConn != nil {
			defer fgaReplicaClientConn.Close()
		}
	}
	aclClient := acl.NewACLClient(fgaClient, fgaReplicaClient, redisClient)

	models := make([]*datamodel.Model, 0, batchSize)
	if err := c.DB.Select("id", "uid", "owner").FindInBatches(&models, batchSize, func(tx *gorm.DB, _ int) error {
		for _, m := range models {
			c.Logger.Log(zap.DebugLevel, fmt.Sprintf("Setting model %s to public", m.ID))
			if err := aclClient.SetPublicModelPermission(ctx, m.UID); err != nil {
				return err
			}
		}
		return nil
	}).Error; err != nil {
		return err
	}

	return nil
}
