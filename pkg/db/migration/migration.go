package migration

import (
	"context"

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

	switch version {
	case 8:
		ctx := context.Background()
		l, _ := logger.GetZapLogger(ctx)

		db := database.GetConnection().WithContext(ctx)
		defer database.Close(db)

		mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient(ctx)
		if mgmtPrivateServiceClientConn != nil {
			defer mgmtPrivateServiceClientConn.Close()
		}

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
