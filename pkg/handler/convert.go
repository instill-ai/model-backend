package handler

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/model-backend/internal/logger"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
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

func DBModelToPBModel(modelDef *datamodel.ModelDefinition, dbModel *datamodel.Model) *modelPB.Model {
	logger, _ := logger.GetZapLogger()

	pbModel := modelPB.Model{
		Name:            fmt.Sprintf("models/%s", dbModel.ID),
		Uid:             dbModel.BaseDynamic.UID.String(),
		Id:              dbModel.ID,
		CreateTime:      timestamppb.New(dbModel.CreateTime),
		UpdateTime:      timestamppb.New(dbModel.UpdateTime),
		Description:     &dbModel.Description,
		ModelDefinition: fmt.Sprintf("model-definitions/%s", modelDef.ID),
		Visibility:      modelPB.Model_Visibility(dbModel.Visibility),
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
					})
					if err != nil {
						logger.Fatal(err.Error())
					}
					if err := str.UnmarshalJSON(b); err != nil {
						logger.Fatal(err.Error())
					}
				} else {
					_ = str.UnmarshalJSON(dbModel.Configuration)
					// if err != nil {
					// 	logger.Fatal(err.Error())
					// }
				}
				return &str
			}
			return nil
		}(),
	}
	if strings.HasPrefix(dbModel.Owner, "users/") {
		userName := resource.GetUserNameByUid(strings.TrimPrefix(dbModel.Owner, "users/"))
		pbModel.Owner = &modelPB.Model_User{User: "users/" + userName}
	} else if strings.HasPrefix(dbModel.Owner, "organizations/") {
		userName := resource.GetUserNameByUid(strings.TrimPrefix(dbModel.Owner, "organizations/"))
		pbModel.Owner = &modelPB.Model_Org{Org: "organizations/" + userName}
	}
	return &pbModel
}

func DBModelInstanceToPBModelInstance(modelDef *datamodel.ModelDefinition, model *datamodel.Model, dbModelInstance *datamodel.ModelInstance) *modelPB.ModelInstance {
	logger, _ := logger.GetZapLogger()

	pbModelInstance := modelPB.ModelInstance{
		Name:            fmt.Sprintf("models/%s/instances/%s", model.ID, dbModelInstance.ID),
		Uid:             dbModelInstance.BaseDynamic.UID.String(),
		Id:              dbModelInstance.ID,
		CreateTime:      timestamppb.New(dbModelInstance.CreateTime),
		UpdateTime:      timestamppb.New(dbModelInstance.UpdateTime),
		ModelDefinition: fmt.Sprintf("model-definitions/%s", modelDef.ID),
		State:           modelPB.ModelInstance_State(dbModelInstance.State),
		Task:            modelPB.ModelInstance_Task(dbModelInstance.Task),
		Configuration: func() *structpb.Struct {
			if dbModelInstance.Configuration != nil {
				str := structpb.Struct{}
				if err := str.UnmarshalJSON(dbModelInstance.Configuration); err != nil {
					logger.Fatal(err.Error())
				}
				return &str
			}
			return nil
		}(),
	}
	return &pbModelInstance
}

func DBModelDefinitionToPBModelDefinition(dbModelDefinition *datamodel.ModelDefinition) *modelPB.ModelDefinition {
	logger, _ := logger.GetZapLogger()

	pbModelDefinition := modelPB.ModelDefinition{
		Name:             fmt.Sprintf("model-definitions/%s", dbModelDefinition.ID),
		Id:               dbModelDefinition.ID,
		Uid:              dbModelDefinition.BaseStatic.UID.String(),
		Title:            dbModelDefinition.Title,
		DocumentationUrl: dbModelDefinition.DocumentationUrl,
		Icon:             dbModelDefinition.Icon,
		CreateTime:       timestamppb.New(dbModelDefinition.CreateTime),
		UpdateTime:       timestamppb.New(dbModelDefinition.UpdateTime),
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
		ModelInstanceSpec: func() *structpb.Struct {
			if dbModelDefinition.ModelInstanceSpec != nil {
				var specification = &structpb.Struct{}
				if err := protojson.Unmarshal([]byte(dbModelDefinition.ModelInstanceSpec.String()), specification); err != nil {
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
