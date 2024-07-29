package migration

import (
	"context"

	database "github.com/instill-ai/model-backend/pkg/db"
)

// type migration interface {
// 	Migrate() error
// }

// Migrate executes custom code as part of a database migration. This code is
// intended to be run only once and typically goes along a change
// in the database schemas. Some use cases might be backfilling a table or
// updating some existing records according to the schema changes.
//
// Note that the changes in the database schemas shouldn't be run here, only
// code accompanying them.
func Migrate(version uint) error {
	ctx := context.Background()

	db := database.GetConnection().WithContext(ctx)
	defer database.Close(db)

	return nil
}
