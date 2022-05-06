package main

import (
	"gorm.io/gorm"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/model-backend/pkg/datamodel"
)

func createModelDefinitionRecord(
	db *gorm.DB,
	id string,
	uid string,
	title string,
	documentationURL string,
	icon string,
	public bool,
	custom bool,
	spec datamodel.Spec,
) error {
	uid_, _ := uuid.FromString(uid)
	modelDef := datamodel.ModelDefinition{
		BaseStatic:       datamodel.BaseStatic{UID: uid_},
		ID:               id,
		DocumentationUrl: documentationURL,
		Icon:             icon,
		Spec:             spec,
		Public:           public,
		Custom:           custom,
		Title:            title,
	}

	if result := db.Model(&datamodel.ModelDefinition{}).Create(&modelDef); result.Error != nil {
		return result.Error
	}

	return nil
}
