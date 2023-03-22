package main

import (
	"encoding/json"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"gorm.io/gorm"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"

	database "github.com/instill-ai/model-backend/pkg/db"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

var enumRegistry = map[string]map[string]int32{
	"release_stage": modelPB.ReleaseStage_value,
}

func createModelDefinition(db *gorm.DB, modelDef *modelPB.ModelDefinition) error {
	modelSpecBytes, _ := json.Marshal(modelDef.GetModelSpec())
	modelInstanceSpecBytes, _ := json.Marshal(modelDef.GetModelInstanceSpec())
	if err := createModelDefinitionRecord(
		db,
		modelDef.GetId(),
		modelDef.GetUid(),
		modelDef.GetTitle(),
		modelDef.GetDocumentationUrl(),
		modelDef.GetIcon(),
		modelSpecBytes,
		modelInstanceSpecBytes,
		datamodel.ReleaseStage(modelDef.GetReleaseStage()),
	); err != nil {
		return err
	}

	return nil
}

func main() {

	logger, _ := logger.GetZapLogger()
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()
	grpc_zap.ReplaceGrpcLoggerV2(logger)

	if err := config.Init(); err != nil {
		logger.Fatal(err.Error())
	}

	db := database.GetConnection()
	defer database.Close(db)

	datamodel.InitJSONSchema()

	modelDefs := []*modelPB.ModelDefinition{}

	if err := loadDefinitions(&modelDefs); err != nil {
		logger.Fatal(err.Error())
	}

	for _, def := range modelDefs {
		// Validate JSON Schema before inserting into db
		if err := datamodel.ValidateJSONSchema(datamodel.ModelDefJSONSchema, def, true); err != nil {
			logger.Fatal(err.Error())
		}
		// Create source definition record
		if err := createModelDefinition(db, def); err != nil {
			logger.Fatal(err.Error())
		}
	}

}
