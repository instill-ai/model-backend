package convert000008

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/instill-ai/model-backend/pkg/acl"
	"github.com/instill-ai/model-backend/pkg/datamodel"

	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

const batchSize = 100

// ModelACLConverter executes code along with the 8th database
// schema revision.
type ModelACLConverter struct {
	DB         *gorm.DB
	Logger     *zap.Logger
	ACLClient  *acl.ACLClient
	MgmtClient mgmtpb.MgmtPrivateServiceClient
}

// Migrate updates all models' visibility to VISIBILITY_PUBLIC
// and migrate the existing owner field to new namespace_id and namespace_type field.
func (c *ModelACLConverter) Migrate() error {
	c.Logger.Info("NamespaceVisibilityAndIDMigrator start")
	if err := c.migrateModel(); err != nil {
		return err
	}
	return nil
}

func (c *ModelACLConverter) migrateModel() error {
	ctx := context.Background()

	models := make([]*datamodel.Model, 0, batchSize)
	if err := c.DB.Select("id", "uid", "owner").FindInBatches(&models, batchSize, func(tx *gorm.DB, _ int) error {
		for _, m := range models {
			l := c.Logger.With(zap.String("modelUID", m.UID.String()))

			l.Info(fmt.Sprintf("Migrating model: %s to public", m.ID))
			if err := c.ACLClient.SetPublicModelPermission(ctx, m.UID); err != nil {
				l.Error(fmt.Sprintf("Migrating model: %s to public failed!!", m.ID))
				return fmt.Errorf("migrating model: %w to public failed", err)
			}

			ownerUID := strings.Split(m.Owner, "/")[1]

			ns, err := c.MgmtClient.CheckNamespaceByUIDAdmin(ctx, &mgmtpb.CheckNamespaceByUIDAdminRequest{
				Uid: ownerUID,
			})
			if err != nil {
				return err
			}

			nsType := ""
			if ns.Type == mgmtpb.CheckNamespaceByUIDAdminResponse_NAMESPACE_ORGANIZATION {
				nsType = "organizations"
			} else if ns.Type == mgmtpb.CheckNamespaceByUIDAdminResponse_NAMESPACE_USER {
				nsType = "users"
			}
			result := tx.Model(m).Where("uid = ?", m.UID).Update("namespace_id", ns.Id).Update("namespace_type", nsType)
			if result.Error != nil {
				l.Error(fmt.Sprintf("Update model: %s namespace failed!!", m.ID))
				return fmt.Errorf("update model: %w namespace failed", result.Error)
			}
		}
		return nil
	}).Error; err != nil {
		return err
	}

	return nil
}
