package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (s *service) PBToDBModel(ctx context.Context, ns resource.Namespace, pbModel *modelPB.Model) *datamodel.Model {
	logger, _ := custom_logger.GetZapLogger(ctx)

	return &datamodel.Model{
		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbModel.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbModel.GetUid())
				if err != nil {
					logger.Fatal(err.Error())
				}
				return id
			}(),
			CreateTime: func() time.Time {
				if pbModel.GetCreateTime() != nil {
					return pbModel.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbModel.GetUpdateTime() != nil {
					return pbModel.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},
		Owner: ns.Permalink(),
		ID:    pbModel.GetId(),
		Description: sql.NullString{
			String: pbModel.GetDescription(),
			Valid:  true,
		},
		Task:             datamodel.ModelTask(pbModel.GetTask()),
		Visibility:       datamodel.ModelVisibility(pbModel.GetVisibility()),
		Region:           pbModel.GetRegion(),
		Hardware:         pbModel.GetHardware(),
		Readme:           pbModel.GetReadme(),
		SourceURL:        pbModel.GetSourceUrl(),
		DocumentationURL: pbModel.GetDocumentationUrl(),
		License:          pbModel.GetLicense(),
	}
}

func (s *service) DBToPBModel(ctx context.Context, modelDef *datamodel.ModelDefinition, dbModel *datamodel.Model) (*modelPB.Model, error) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	ownerName, err := s.ConvertOwnerPermalinkToName(dbModel.Owner)
	if err != nil {
		return nil, err
	}
	owner, err := s.FetchOwnerWithPermalink(dbModel.Owner)
	if err != nil {
		return nil, err
	}

	pbModel := modelPB.Model{
		Name:       fmt.Sprintf("%s/models/%s", ownerName, dbModel.ID),
		Uid:        dbModel.BaseDynamic.UID.String(),
		Id:         dbModel.ID,
		CreateTime: timestamppb.New(dbModel.CreateTime),
		UpdateTime: timestamppb.New(dbModel.UpdateTime),
		DeleteTime: func() *timestamppb.Timestamp {
			if dbModel.DeleteTime.Time.IsZero() {
				return nil
			} else {
				return timestamppb.New(dbModel.DeleteTime.Time)
			}
		}(),
		Description:     &dbModel.Description.String,
		ModelDefinition: fmt.Sprintf("model-definitions/%s", modelDef.ID),
		Visibility:      modelPB.Model_Visibility(dbModel.Visibility),
		Task:            commonPB.Task(dbModel.Task),
		Configuration: func() *structpb.Struct {
			if dbModel.Configuration != nil {
				str := structpb.Struct{}
				err := str.UnmarshalJSON(dbModel.Configuration)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return &str
			}
			return nil
		}(),
		OwnerName:        ownerName,
		Owner:            owner,
		Region:           dbModel.Region,
		Hardware:         dbModel.Hardware,
		SourceUrl:        dbModel.SourceURL,
		DocumentationUrl: dbModel.DocumentationURL,
		License:          dbModel.License,
	}

	return &pbModel, nil
}

func (s *service) DBToPBModels(ctx context.Context, dbModels []*datamodel.Model) ([]*modelPB.Model, error) {

	pbModels := make([]*modelPB.Model, len(dbModels))

	for idx := range dbModels {
		modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModels[idx].ModelDefinitionUID)
		if err != nil {
			return nil, err
		}

		pbModels[idx], err = s.DBToPBModel(
			ctx,
			modelDef,
			dbModels[idx],
		)
		if err != nil {
			return nil, err
		}
	}

	return pbModels, nil
}

func (s *service) DBToPBModelDefinition(ctx context.Context, dbModelDefinition *datamodel.ModelDefinition) (*modelPB.ModelDefinition, error) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	pbModelDefinition := modelPB.ModelDefinition{
		Name:             fmt.Sprintf("model-definitions/%s", dbModelDefinition.ID),
		Id:               dbModelDefinition.ID,
		Uid:              dbModelDefinition.BaseStatic.UID.String(),
		Title:            dbModelDefinition.Title,
		DocumentationUrl: dbModelDefinition.DocumentationURL,
		Icon:             dbModelDefinition.Icon,
		CreateTime:       timestamppb.New(dbModelDefinition.CreateTime),
		UpdateTime:       timestamppb.New(dbModelDefinition.UpdateTime),
		ReleaseStage:     modelPB.ReleaseStage(dbModelDefinition.ReleaseStage),
		ModelSpec: func() *structpb.Struct {
			if dbModelDefinition.ModelSpec != nil {
				var specification = &structpb.Struct{}
				if err := protojson.Unmarshal([]byte(dbModelDefinition.ModelSpec.String()), specification); err != nil {
					logger.Fatal(err.Error())
				}
				return specification
			} else {
				return nil
			}
		}(),
	}

	return &pbModelDefinition, nil
}

func (s *service) DBToPBModelDefinitions(ctx context.Context, dbModelDefinitions []*datamodel.ModelDefinition) ([]*modelPB.ModelDefinition, error) {

	var err error
	pbModelDefinitions := make([]*modelPB.ModelDefinition, len(dbModelDefinitions))

	for idx := range dbModelDefinitions {
		pbModelDefinitions[idx], err = s.DBToPBModelDefinition(
			ctx,
			dbModelDefinitions[idx],
		)
		if err != nil {
			return nil, err
		}
	}

	return pbModelDefinitions, nil
}
