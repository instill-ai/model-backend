package main

import (
	"gorm.io/gorm"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/pkg/datamodel"
)

func createModelDefinitionRecord(
	db *gorm.DB,
	uid string,
	title string,
	documentationURL string,
	icon string,
	public bool,
	custom bool,
	spec datamodel.Spec,
) error {
	id, _ := uuid.FromString(uid)
	modelDef := datamodel.ModelDefinition{
		BaseStatic:       datamodel.BaseStatic{UID: id},
		DocumentationUrl: documentationURL,
		Icon:             icon,
		Spec:             spec,
		Public:           public,
		Custom:           custom,
	}

	if result := db.Model(&datamodel.ModelDefinition{}).Create(&modelDef); result.Error != nil {
		return result.Error
	}

	return nil
}
