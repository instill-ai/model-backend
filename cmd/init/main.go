package main

import (
	"encoding/json"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	"gorm.io/gorm"

	"github.com/instill-ai/model-backend/configs"
	"github.com/instill-ai/model-backend/internal/logger"

	database "github.com/instill-ai/model-backend/internal/db"
	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

func createModelDefinition(db *gorm.DB, modelDef *modelPB.ModelDefinition) error {
	modelSpecBytes, _ := json.Marshal(modelDef.GetModelSpec())
	modelInstanceSpecBytes, _ := json.Marshal(modelDef.GetModelSpec())
	if err := createModelDefinitionRecord(
		db,
		modelDef.GetId(),
		modelDef.GetUid(),
		modelDef.GetTitle(),
		modelDef.GetDocumentationUrl(),
		modelDef.GetIcon(),
		modelSpecBytes,
		modelInstanceSpecBytes,
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

	if err := configs.Init(); err != nil {
		logger.Fatal(err.Error())
	}

	db := database.GetConnection()
	defer database.Close(db)

	modelDefs := []*modelPB.ModelDefinition{}

	if err := loadDefinitions(&modelDefs); err != nil {
		logger.Fatal(err.Error())
	}

	for _, def := range modelDefs {
		// Create source definition record
		if err := createModelDefinition(db, def); err != nil {
			logger.Fatal(err.Error())
		}
	}

}
