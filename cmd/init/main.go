package main

import (
	"context"
	"encoding/json"
	"log"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/datamodel"

	database "github.com/instill-ai/model-backend/pkg/db"
	databaseInit "github.com/instill-ai/model-backend/pkg/init"
	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func createModelDefinition(db *gorm.DB, modelDef *modelPB.ModelDefinition) error {
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

	// setup tracing
	ctx, cancel := context.WithCancel(context.Background())

	if tp, err := custom_otel.SetupTracing(ctx, "model-backend-init"); err != nil {
		panic(err)
	} else {
		defer func() {
			err = tp.Shutdown(ctx)
		}()
	}

	ctx, span := otel.Tracer("init-tracer").Start(ctx,
		"main",
	)
	defer span.End()
	defer cancel()

	logger, _ := custom_logger.GetZapLogger(ctx)

	db := database.GetConnection()
	defer database.Close(db)

	datamodel.InitJSONSchema(ctx)

	modelDefs := []*modelPB.ModelDefinition{}

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
