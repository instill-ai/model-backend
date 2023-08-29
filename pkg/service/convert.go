package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/logger"

	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func (s *service) PBToDBModel(ctx context.Context, pbModel *modelPB.Model) *datamodel.Model {
	logger, _ := logger.GetZapLogger(ctx)

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
		ID: pbModel.GetId(),
		Description: sql.NullString{
			String: pbModel.GetDescription(),
			Valid:  true,
		},
	}
}

func (s *service) DBToPBModel(ctx context.Context, modelDef *datamodel.ModelDefinition, dbModel *datamodel.Model) (*modelPB.Model, error) {
	logger, _ := logger.GetZapLogger(ctx)

	owner, err := s.ConvertOwnerPermalinkToName(dbModel.Owner)
	if err != nil {
		return nil, err
	}

	pbModel := modelPB.Model{
		Name:            fmt.Sprintf("%s/models/%s", owner, dbModel.ID),
		Uid:             dbModel.BaseDynamic.UID.String(),
		Id:              dbModel.ID,
		CreateTime:      timestamppb.New(dbModel.CreateTime),
		UpdateTime:      timestamppb.New(dbModel.UpdateTime),
		Description:     &dbModel.Description.String,
		ModelDefinition: fmt.Sprintf("model-definitions/%s", modelDef.ID),
		Visibility:      modelPB.Model_Visibility(dbModel.Visibility),
		State:           modelPB.Model_State(dbModel.State),
		Task:            commonPB.Task(dbModel.Task),
		Configuration: func() *structpb.Struct {
			if dbModel.Configuration != nil {
				str := structpb.Struct{}
				// remove credential in ArtiVC model configuration
				if modelDef.ID == "artivc" {
					var modelConfig datamodel.ArtiVCModelConfiguration
					if err := json.Unmarshal([]byte(dbModel.Configuration.String()), &modelConfig); err != nil {
						logger.Fatal(err.Error())
					}
					b, err := json.Marshal(&datamodel.ArtiVCModelConfiguration{
						Url: modelConfig.Url,
						Tag: modelConfig.Tag,
					})
					if err != nil {
						logger.Fatal(err.Error())
					}
					if err := str.UnmarshalJSON(b); err != nil {
						logger.Fatal(err.Error())
					}
				} else {
					err := str.UnmarshalJSON(dbModel.Configuration)
					if err != nil {
						logger.Fatal(err.Error())
					}
				}
				return &str
			}
			return nil
		}(),
	}

	if strings.HasPrefix(dbModel.Owner, "users/") {
		pbModel.Owner = &modelPB.Model_User{User: dbModel.Owner}
	} else if strings.HasPrefix(dbModel.Owner, "orgs/") {
		pbModel.Owner = &modelPB.Model_Org{Org: dbModel.Owner}
	}
	return &pbModel, nil
}

func (s *service) DBToPBModels(ctx context.Context, dbModels []*datamodel.Model) ([]*modelPB.Model, error) {

	pbModels := make([]*modelPB.Model, len(dbModels))

	for idx := range dbModels {
		modelDef, err := s.GetRepository().GetModelDefinitionByUID(dbModels[idx].ModelDefinitionUid)
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
	logger, _ := logger.GetZapLogger(ctx)

	pbModelDefinition := modelPB.ModelDefinition{
		Name:             fmt.Sprintf("model-definitions/%s", dbModelDefinition.ID),
		Id:               dbModelDefinition.ID,
		Uid:              dbModelDefinition.BaseStatic.UID.String(),
		Title:            dbModelDefinition.Title,
		DocumentationUrl: dbModelDefinition.DocumentationUrl,
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
