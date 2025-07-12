package migration

import (
	"context"

	"go.uber.org/zap"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/db/migration/convert/convert000008"

	database "github.com/instill-ai/model-backend/pkg/db"
	grpcclientx "github.com/instill-ai/x/client/grpc"
	logx "github.com/instill-ai/x/log"
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

	switch version {
	case 8:
		ctx := context.Background()
		l, _ := logx.GetZapLogger(ctx)

		db := database.GetConnection().WithContext(ctx)
		defer database.Close(db)

		mgmtPrivateServiceClient, mclose, err := grpcclientx.NewMgmtPrivateClient(config.Config.MgmtBackend)
		if err != nil {
			l.Fatal("failed to create mgmt private service client", zap.Error(err))
		}
		defer func() {
			err = mclose()
			if err != nil {
				l.Fatal("failed to close mgmt private service client", zap.Error(err))
			}
		}()

		m = &convert000008.ModelACLConverter{
			DB:         db,
			Logger:     l,
			MgmtClient: mgmtPrivateServiceClient,
		}
	default:
		return nil
	}

	return m.Migrate()
}
