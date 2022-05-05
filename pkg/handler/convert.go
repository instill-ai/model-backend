package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/pkg/datamodel"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
)

func PBModelToDBModel(owner string, pbModel *modelPB.Model) *datamodel.Model {
	logger, _ := logger.GetZapLogger()

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
		ID:          pbModel.GetId(),
		Description: pbModel.GetDescription(),
	}
}

func DBModelToPBModel(dbModel *datamodel.Model) *modelPB.Model {
	logger, _ := logger.GetZapLogger()

	pbModel := modelPB.Model{
		Name:            fmt.Sprintf("models/%s", dbModel.ID),
		Uid:             dbModel.BaseDynamic.UID.String(),
		Id:              dbModel.ID,
		CreateTime:      timestamppb.New(dbModel.CreateTime),
		UpdateTime:      timestamppb.New(dbModel.UpdateTime),
		Description:     &dbModel.Description,
		ModelDefinition: dbModel.ModelDefinition.String(),
		Visibility:      modelPB.Model_Visibility(dbModel.Visibility),
		Configuration: func() *modelPB.Spec {
			if dbModel.Configuration.Specification != nil {
				var specification = &structpb.Struct{}
				err := protojson.Unmarshal([]byte(dbModel.Configuration.Specification.String()), specification)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return &modelPB.Spec{
					DocumentationUrl: dbModel.Configuration.DocumentationUrl,
					Specification:    specification,
				}
			} else {
				return &modelPB.Spec{
					DocumentationUrl: dbModel.Configuration.DocumentationUrl,
				}
			}
		}(),
	}
	if strings.HasPrefix(dbModel.Owner, "users/") {
		pbModel.Owner = &modelPB.Model_User{User: dbModel.Owner}
	} else if strings.HasPrefix(dbModel.Owner, "organizations/") {
		pbModel.Owner = &modelPB.Model_Org{Org: dbModel.Owner}
	}
	return &pbModel
}

func DBModelInstanceToPBModelInstance(modelId string, dbModelInstance *datamodel.ModelInstance) *modelPB.ModelInstance {
	logger, _ := logger.GetZapLogger()

	pbModelInstance := modelPB.ModelInstance{
		Name:            fmt.Sprintf("models/%s/instances/%s", modelId, dbModelInstance.ID),
		Uid:             dbModelInstance.BaseDynamic.UID.String(),
		Id:              dbModelInstance.ID,
		CreateTime:      timestamppb.New(dbModelInstance.CreateTime),
		UpdateTime:      timestamppb.New(dbModelInstance.UpdateTime),
		ModelDefinition: dbModelInstance.ModelDefinition,
		State:           modelPB.ModelInstance_State(dbModelInstance.State),
		Task:            modelPB.ModelInstance_Task(dbModelInstance.Task),
		Configuration: func() *modelPB.Spec {
			if dbModelInstance.Configuration.Specification != nil {
				var specification = &structpb.Struct{}
				err := protojson.Unmarshal([]byte(dbModelInstance.Configuration.Specification.String()), specification)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return &modelPB.Spec{
					DocumentationUrl: dbModelInstance.Configuration.DocumentationUrl,
					Specification:    specification,
				}
			} else {
				return &modelPB.Spec{
					DocumentationUrl: dbModelInstance.Configuration.DocumentationUrl,
				}
			}
		}(),
	}

	return &pbModelInstance
}

func DBModelDefinitionToPBModelDefinition(dbModelDefinition *datamodel.ModelDefinition) *modelPB.ModelDefinition {
	logger, _ := logger.GetZapLogger()

	pbModelDefinition := modelPB.ModelDefinition{
		Name:             fmt.Sprintf("model-definitions/%s", dbModelDefinition.UID),
		Uid:              dbModelDefinition.BaseStatic.UID.String(),
		Title:            dbModelDefinition.Title,
		DocumentationUrl: dbModelDefinition.Spec.DocumentationUrl,
		Icon:             dbModelDefinition.Icon,
		Public:           dbModelDefinition.Public,
		Custom:           dbModelDefinition.Custom,
		CreateTime:       timestamppb.New(dbModelDefinition.CreateTime),
		UpdateTime:       timestamppb.New(dbModelDefinition.UpdateTime),
		Spec: func() *modelPB.Spec {
			if dbModelDefinition.Spec.Specification != nil {
				var specification = &structpb.Struct{}
				err := protojson.Unmarshal([]byte(dbModelDefinition.Spec.Specification.String()), specification)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return &modelPB.Spec{
					DocumentationUrl: dbModelDefinition.Spec.DocumentationUrl,
					Specification:    specification,
				}
			} else {
				return &modelPB.Spec{
					DocumentationUrl: dbModelDefinition.Spec.DocumentationUrl,
				}
			}
		}(),
	}

	return &pbModelDefinition
}
