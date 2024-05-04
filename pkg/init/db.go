package init

import (
	"gorm.io/gorm"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/pkg/datamodel"
)

func CreateModelDefinitionRecord(
	db *gorm.DB,
	id string,
	uid string,
	title string,
	documentationURL string,
	icon string,
	modelSpec []byte,
	releaseStage datamodel.ReleaseStage,
) error {
	uid_, _ := uuid.FromString(uid)
	modelDef := datamodel.ModelDefinition{
		BaseStatic:       datamodel.BaseStatic{UID: uid_},
		ID:               id,
		DocumentationURL: documentationURL,
		Icon:             icon,
		ModelSpec:        modelSpec,
		Title:            title,
		ReleaseStage:     releaseStage,
	}

	if result := db.Model(&datamodel.ModelDefinition{}).FirstOrCreate(&modelDef); result.Error != nil {
		return result.Error
	}

	return nil
}
