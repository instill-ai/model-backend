package migration

import (
	"context"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	openfga "github.com/openfga/api/proto/openfga/v1"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/db/migration/convert/convert000008"
	"github.com/instill-ai/model-backend/pkg/external"
	"github.com/instill-ai/model-backend/pkg/logger"

	database "github.com/instill-ai/model-backend/pkg/db"
)

type migration interface {
	Migrate() error
}

// Migrate executes custom code as part of a database migration. This code is
// intended to be run only once and typically goes along a change
// in the database schemas. Some use cases might be backfilling a table or
// updating some existing records according to the schema changes.
//
// Note that the changes in the database schemas shouldn't be run here, only
// code accompanying them.
func Migrate(version uint) error {
	var m migration
	ctx := context.Background()
	l, _ := logger.GetZapLogger(ctx)

	db := database.GetConnection().WithContext(ctx)
	defer database.Close(db)

	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient(ctx)
	if mgmtPrivateServiceClientConn != nil {
		defer mgmtPrivateServiceClientConn.Close()
	}

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

	switch version {
	case 8:
		m = &convert000008.ModelACLConverter{
			DB:         db,
			Logger:     l,
			MgmtClient: mgmtPrivateServiceClient,
			ACLClient:  &aclClient,
		}
	default:
		return nil
	}

	return m.Migrate()
}
