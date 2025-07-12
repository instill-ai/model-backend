package main

import (
	"context"
	"encoding/json"
	"log"

	"gorm.io/gorm"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"

	database "github.com/instill-ai/model-backend/pkg/db"
	databaseInit "github.com/instill-ai/model-backend/pkg/init"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	logx "github.com/instill-ai/x/log"
)

func createModelDefinition(db *gorm.DB, modelDef *modelpb.ModelDefinition) error {
	modelSpecBytes, _ := json.Marshal(modelDef.GetModelSpec())
	if err := databaseInit.CreateModelDefinitionRecord(
		db,
		modelDef.GetId(),
		modelDef.GetUid(),
		modelDef.GetTitle(),
		modelDef.GetDocumentationUrl(),
		modelDef.GetIcon(),
		modelSpecBytes,
		datamodel.ReleaseStage(modelDef.GetReleaseStage()),
	); err != nil {
		return err
	}

	return nil
}

func main() {

	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	logx.Debug = config.Config.Server.Debug
	logger, _ := logx.GetZapLogger(context.Background())
	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()

	// Set gRPC logging based on debug mode
	if config.Config.Server.Debug {
		grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 0) // All logs
	} else {
		grpczap.ReplaceGrpcLoggerV2WithVerbosity(logger, 3) // verbosity 3 will avoid [transport] from emitting
	}

	// Initialize OpenTelemetry tracing
	db := database.GetConnection()
	defer database.Close(db)

	datamodel.InitJSONSchema(context.Background())

	modelDefs := []*modelpb.ModelDefinition{}

	if err := databaseInit.LoadDefinitions(&modelDefs); err != nil {
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
