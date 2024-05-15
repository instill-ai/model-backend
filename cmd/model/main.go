package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

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
	Region          string         `json:"region"`
	Hardwdare       string         `json:"hardware"`
	Configuration   map[string]any `json:"configuration"`
	Version         string         `json:"version"`
}

type GetNamespaceModelResponseInterface interface {
	GetModel() *modelPB.Model
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

// InitModelPrivateServiceClient initializes a ModelServiceClient instance
func InitModelPrivateServiceClient(ctx context.Context) (modelPB.ModelPrivateServiceClient, *grpc.ClientConn) {
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
	clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", host, config.Config.Server.PrivatePort), clientDialOpts)
	if err != nil {
		logger.Error(err.Error())
		return nil, nil
	}

	return modelPB.NewModelPrivateServiceClient(clientConn), clientConn
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

	if err := config.Init(config.ParseConfigFlag()); err != nil {
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

	modelPrivateServiceClient, modelPrivateServiceClientConn := InitModelPrivateServiceClient(ctx)
	if modelPrivateServiceClientConn != nil {
		defer modelPrivateServiceClientConn.Close()
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

			if config.Config.InitModel.OwnerType == string(resource.User) {
				_, err = modelPublicServiceClient.GetUserModel(ctx, &modelPB.GetUserModelRequest{
					Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
					View: modelPB.View_VIEW_FULL.Enum(),
				})
			} else if config.Config.InitModel.OwnerType == string(resource.Organization) {
				_, err = modelPublicServiceClient.GetOrganizationModel(ctx, &modelPB.GetOrganizationModelRequest{
					Name: fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
					View: modelPB.View_VIEW_FULL.Enum(),
				})
			}
			if err != nil {
				logger.Info("Creating model: " + modelConfig.ID)

				model := &modelPB.Model{
					Id:              modelConfig.ID,
					Description:     &modelConfig.Description,
					ModelDefinition: modelConfig.ModelDefinition,
					Visibility:      modelPB.Model_VISIBILITY_PUBLIC,
					Region:          modelConfig.Region,
					Hardware:        modelConfig.Hardwdare,
					Configuration:   configuration,
				}
				if config.Config.InitModel.OwnerType == string(resource.User) {
					_, err = modelPublicServiceClient.CreateUserModel(ctx, &modelPB.CreateUserModelRequest{
						Model:  model,
						Parent: name,
					})
				} else if config.Config.InitModel.OwnerType == string(resource.Organization) {
					_, err = modelPublicServiceClient.CreateOrganizationModel(ctx, &modelPB.CreateOrganizationModelRequest{
						Model:  model,
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
				}
			}
			logger.Info("Deploying model: " + modelConfig.ID)
			_, err = modelPrivateServiceClient.DeployModelAdmin(ctx, &modelPB.DeployModelAdminRequest{
				Name:    fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
				Version: modelConfig.Version,
			})
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
			state := modelPB.State_STATE_OFFLINE
			var message string
			for state != modelPB.State_STATE_ACTIVE {
				time.Sleep(2 * time.Second)
				if config.Config.InitModel.OwnerType == string(resource.User) {
					var resp *modelPB.WatchUserModelResponse
					resp, err = modelPublicServiceClient.WatchUserModel(ctx, &modelPB.WatchUserModelRequest{
						Name:    fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
						Version: modelConfig.Version,
					})
					state = resp.GetState()
					message = resp.GetMessage()
				} else if config.Config.InitModel.OwnerType == string(resource.Organization) {
					var resp *modelPB.WatchOrganizationModelResponse
					resp, err = modelPublicServiceClient.WatchOrganizationModel(ctx, &modelPB.WatchOrganizationModelRequest{
						Name:    fmt.Sprintf("%s/models/%s", name, modelConfig.ID),
						Version: modelConfig.Version,
					})
					state = resp.GetState()
					message = resp.GetMessage()
				}
				if err != nil {
					logger.Info(fmt.Sprintf("Deploy model err: %v", err))
					if e, ok := status.FromError(err); ok {
						if e.Code() != codes.AlreadyExists {
							logger.Fatal("handler.DeployModelAdmin: " + err.Error())
							return
						}
						return
					}
					return
				}
				logger.Info(fmt.Sprintf("%s: %v, message: %s", modelConfig.ID, state, message))
			}
		}(modelConfig)
	}

	wg.Wait()

	logger.Info("creating and deploying models done!")
	span.End()
}
