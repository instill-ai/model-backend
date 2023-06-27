package handler

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

	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func PBModelToDBModel(ctx context.Context, owner string, pbModel *modelPB.Model) *datamodel.Model {
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

func DBModelToPBModel(ctx context.Context, modelDef *datamodel.ModelDefinition, dbModel *datamodel.Model, ownerName string) *modelPB.Model {
	logger, _ := logger.GetZapLogger(ctx)

	pbModel := modelPB.Model{
		Name:            fmt.Sprintf("models/%s", dbModel.ID),
		Uid:             dbModel.BaseDynamic.UID.String(),
		Id:              dbModel.ID,
		CreateTime:      timestamppb.New(dbModel.CreateTime),
		UpdateTime:      timestamppb.New(dbModel.UpdateTime),
		Description:     &dbModel.Description.String,
		ModelDefinition: fmt.Sprintf("model-definitions/%s", modelDef.ID),
		Visibility:      modelPB.Model_Visibility(dbModel.Visibility),
		State:           modelPB.Model_State(dbModel.State),
		Task:            modelPB.Model_Task(dbModel.Task),
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

	if strings.HasPrefix(ownerName, "users/") {
		pbModel.Owner = &modelPB.Model_User{User: ownerName}
	} else if strings.HasPrefix(ownerName, "organizations/") {
		pbModel.Owner = &modelPB.Model_Org{Org: ownerName}
	}
	return &pbModel
}

func DBModelDefinitionToPBModelDefinition(ctx context.Context, dbModelDefinition *datamodel.ModelDefinition) *modelPB.ModelDefinition {
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

	return &pbModelDefinition
}
