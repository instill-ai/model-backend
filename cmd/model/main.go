package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"github.com/gogo/status"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/internal/resource"
	"github.com/instill-ai/model-backend/pkg/datamodel"
	"github.com/instill-ai/model-backend/pkg/middleware"
	"github.com/instill-ai/model-backend/pkg/utils"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	mgmtPB "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type ModelConfig struct {
	ID              string         `json:"id"`
	Description     string         `json:"description"`
	Task            string         `json:"task"`
	ModelDefinition string         `json:"model_definition"`
	Configuration   map[string]any `json:"configuration"`
}

type GetNamespaceModelResponseInterface interface {
	GetModel() *modelPB.Model
}

type CreateNamespaceModelResponseInterface interface {
	GetOperation() *longrunningpb.Operation
}

// InitMgmtPrivateServiceClient initializes a MgmtPrivateServiceClient instance
func InitMgmtPrivateServiceClient(ctx context.Context) (mgmtPB.MgmtPrivateServiceClient, *grpc.ClientConn) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	var creds credentials.TransportCredentials
	var err error
	if config.Config.MgmtBackend.HTTPS.Cert != "" && config.Config.MgmtBackend.HTTPS.Key != "" {
		creds, err = credentials.NewServerTLSFromFile(config.Config.MgmtBackend.HTTPS.Cert, config.Config.MgmtBackend.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return mgmtPB.NewMgmtPrivateServiceClient(clientConn), clientConn
}

// InitModelPublicServiceClient initializes a ModelServiceClient instance
func InitModelPublicServiceClient(ctx context.Context) (modelPB.ModelPublicServiceClient, *grpc.ClientConn) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	var clientDialOpts grpc.DialOption
	if config.Config.Server.HTTPS.Cert != "" && config.Config.Server.HTTPS.Key != "" {
		creds, err := credentials.NewServerTLSFromFile(config.Config.Server.HTTPS.Cert, config.Config.Server.HTTPS.Key)
		if err != nil {
			logger.Fatal(err.Error())
		}
		clientDialOpts = grpc.WithTransportCredentials(creds)
	} else {
		clientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
	}

	host := os.Getenv("MODEL_BACKEND_HOST")
	if host == "" {
		host = "localhost"
	}
	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", host, config.Config.Server.PublicPort), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return modelPB.NewModelPublicServiceClient(clientConn), clientConn
}

func main() {

	// setup tracing
	ctx, cancel := context.WithTimeout(context.Background(), 3600*time.Second)
	defer cancel()

	if tp, err := custom_otel.SetupTracing(ctx, "model-backend"); err != nil {
		panic(err)
	} else {
		defer func() {
			err = tp.Shutdown(ctx)
		}()
	}

	ctx, span := otel.Tracer("main-tracer").Start(ctx,
		"main",
	)

	logger, _ := custom_logger.GetZapLogger(ctx)

	defer func() {
		// can't handle the error due to https://github.com/uber-go/zap/issues/880
		_ = logger.Sync()
	}()
	grpc_zap.ReplaceGrpcLoggerV2(logger)

	if err := config.Init(); err != nil {
		logger.Fatal(err.Error())
	}

	if !config.Config.InitModel.Enabled {
		return
	}

	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := InitMgmtPrivateServiceClient(ctx)
	if mgmtPrivateServiceClientConn != nil {
		defer mgmtPrivateServiceClientConn.Close()
	}

	modelPublicServiceClient, modelPublicServiceClientConn := InitModelPublicServiceClient(ctx)
	if modelPublicServiceClientConn != nil {
		defer modelPublicServiceClientConn.Close()
	}

	var owner *mgmtPB.User
	name := fmt.Sprintf("%s/%s", config.Config.InitModel.OwnerType, config.Config.InitModel.OwnerID)
	if config.Config.InitModel.OwnerType == string(resource.User) {
		resp, err := mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{
			Name: name,
		})
		if err != nil {
			logger.Fatal(err.Error())
		}
		owner = resp.GetUser()
	} else if config.Config.InitModel.OwnerType == string(resource.Organization) {
		resp, err := mgmtPrivateServiceClient.GetOrganizationAdmin(ctx, &mgmtPB.GetOrganizationAdminRequest{
			Name: name,
		})
		if err != nil {
			logger.Fatal(err.Error())
		}
		owner = resp.GetOrganization().GetOwner()
	}

	ctx = middleware.InjectOwnerToContext(ctx, owner)

	var modelConfigs []ModelConfig
	err := utils.GetJSON(config.Config.InitModel.Path, &modelConfigs)
	if err != nil {
		logger.Fatal(err.Error())
	}
	logger.Info("Creating models ...")
	var wg sync.WaitGroup

	wg.Add(len(modelConfigs))
	for _, modelConfig := range modelConfigs {

		go func(modelConfig ModelConfig) {

			defer wg.Done()
			configuration, err := structpb.NewStruct(modelConfig.Configuration)
			if err != nil {
				log.Fatal("structpb.NewValue: ", err)
				return
			}

			var getResp GetNamespaceModelResponseInterface
			if config.Config.InitModel.OwnerType == string(resource.User) {
				getResp, err = modelPublicServiceClient.GetUserModel(ctx, &modelPB.GetUserModelRequest{
					Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
					View: modelPB.View_VIEW_FULL.Enum(),
				})
			} else if config.Config.InitModel.OwnerType == string(resource.Organization) {
				getResp, err = modelPublicServiceClient.GetOrganizationModel(ctx, &modelPB.GetOrganizationModelRequest{
					Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
					View: modelPB.View_VIEW_FULL.Enum(),
				})
			}

			if err == nil {
				var existedModelConfig datamodel.GitHubModelConfiguration
				b, err := getResp.GetModel().GetConfiguration().MarshalJSON()
				if err != nil {
					logger.Error(fmt.Sprintf("marshal existing model config json err: %v", err))
					return
				}
				if err := json.Unmarshal(b, &existedModelConfig); err != nil {
					logger.Error(fmt.Sprintf("unmarshal existing model config err: %v", err))
					return
				}
				if existedModelConfig.Repository != modelConfig.Configuration["repository"] || existedModelConfig.Tag != modelConfig.Configuration["tag"] {
					logger.Info(fmt.Sprintf("requested repo: %s or tag: %s does not match the existing repo: %v or tag: %v, redeploying...",
						modelConfig.Configuration["repository"],
						modelConfig.Configuration["tag"],
						existedModelConfig.Repository,
						existedModelConfig.Tag))
					if config.Config.InitModel.OwnerType == string(resource.User) {
						_, err = modelPublicServiceClient.DeleteUserModel(ctx, &modelPB.DeleteUserModelRequest{
							Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
						})
					} else if config.Config.InitModel.OwnerType == string(resource.Organization) {
						_, err = modelPublicServiceClient.DeleteOrganizationModel(ctx, &modelPB.DeleteOrganizationModelRequest{
							Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
						})
					}
					if err != nil {
						logger.Error(fmt.Sprintf("delete existing model err: %v", err))
						return
					}
				} else {
					logger.Info("model already existed")
					return
				}
			}

			logger.Info("Creating model: " + modelConfig.ID)
			var createResp CreateNamespaceModelResponseInterface
			if config.Config.InitModel.OwnerType == string(resource.User) {
				createResp, err = modelPublicServiceClient.CreateUserModel(ctx, &modelPB.CreateUserModelRequest{
					Model: &modelPB.Model{
						Id:              modelConfig.ID,
						Description:     &modelConfig.Description,
						Task:            utils.Tasks[strings.ToUpper(modelConfig.Task)],
						ModelDefinition: modelConfig.ModelDefinition,
						Configuration:   configuration,
					},
					Parent: name,
				})
			} else if config.Config.InitModel.OwnerType == string(resource.Organization) {
				createResp, err = modelPublicServiceClient.CreateOrganizationModel(ctx, &modelPB.CreateOrganizationModelRequest{
					Model: &modelPB.Model{
						Id:              modelConfig.ID,
						Description:     &modelConfig.Description,
						Task:            utils.Tasks[strings.ToUpper(modelConfig.Task)],
						ModelDefinition: modelConfig.ModelDefinition,
						Configuration:   configuration,
					},
					Parent: name,
				})
			}
			if err != nil {
				logger.Info(fmt.Sprintf("Created model err: %v", err))
				if e, ok := status.FromError(err); ok {
					if e.Code() != codes.AlreadyExists {
						logger.Fatal("handler.CreateModel: " + err.Error())
						return
					}
					return
				}
				return
			} else {
				isCreated := false
				startTime := time.Now()
				for {
					if isCreated || time.Since(startTime) > 5*time.Minute {
						break
					}
					operation, err := modelPublicServiceClient.GetModelOperation(ctx, &modelPB.GetModelOperationRequest{
						Name: createResp.GetOperation().GetName(),
					})
					if err != nil {
						logger.Fatal("handler.GetModelOperation: " + err.Error())
						return
					}
					isCreated = operation.Operation.Done
					time.Sleep(1 * time.Second)
				}
				if !isCreated {
					logger.Fatal("handler.CreateModel: " + err.Error())
					return
				} else {
					if config.Config.InitModel.OwnerType == string(resource.User) {
						if _, err = modelPublicServiceClient.PublishUserModel(ctx, &modelPB.PublishUserModelRequest{
							Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
						}); err != nil {
							logger.Error(fmt.Sprintf("publish model err: %v", err))
							return
						}
						_, err = modelPublicServiceClient.DeployUserModel(ctx, &modelPB.DeployUserModelRequest{
							Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
						})
					} else if config.Config.InitModel.OwnerType == string(resource.Organization) {
						if _, err = modelPublicServiceClient.PublishOrganizationModel(ctx, &modelPB.PublishOrganizationModelRequest{
							Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
						}); err != nil {
							logger.Error(fmt.Sprintf("publish model err: %v", err))
							return
						}
						_, err = modelPublicServiceClient.DeployOrganizationModel(ctx, &modelPB.DeployOrganizationModelRequest{
							Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
						})
					}
					if err != nil {
						logger.Error(fmt.Sprintf("deploy model err: %v", err))
						if e, ok := status.FromError(err); ok {
							if e.Code() == codes.FailedPrecondition {
								logger.Error(fmt.Sprintf("FailedPrecondition deploy err: %v", e))
								return
							}
							logger.Error(fmt.Sprintf("deploy model err: %v", e))
							return
						}
						logger.Error("handler.DeployModel: " + err.Error())
						return
					}
				}
				return
			}
		}(modelConfig)
	}

	wg.Wait()

	logger.Info("creating and deploying models done!")
	span.End()
}
