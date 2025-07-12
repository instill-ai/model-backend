package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/gogo/status"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/resource"
	"github.com/instill-ai/model-backend/pkg/utils"
	"github.com/instill-ai/x/client"
	"github.com/instill-ai/x/server/grpc/gateway"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	grpcclientx "github.com/instill-ai/x/client/grpc"
	logx "github.com/instill-ai/x/log"
)

// ModelConfig related to model config
type ModelConfig struct {
	ID              string         `json:"id"`
	OwnerType       string         `json:"owner_type"`
	OwnerID         string         `json:"owner_id"`
	Description     string         `json:"description"`
	Task            string         `json:"task"`
	ModelDefinition string         `json:"model_definition"`
	Region          string         `json:"region"`
	Hardware        string         `json:"hardware"`
	Configuration   map[string]any `json:"configuration"`
	Version         string         `json:"version"`
}

func main() {

	if err := config.Init(config.ParseConfigFlag()); err != nil {
		log.Fatal(err.Error())
	}

	if !config.Config.InitModel.Enabled {
		return
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

	// Initialize all clients
	mgmtPrivateServiceClient, modelPublicServiceClient, modelPrivateServiceClient, closeClients := newClients(logger)
	defer closeClients()

	var modelConfigs []ModelConfig
	err := utils.GetJSON(config.Config.InitModel.Path, &modelConfigs)
	if err != nil {
		logger.Fatal(err.Error())
	}

	logger.Info("Creating models ...")

	var wg sync.WaitGroup
	wg.Add(len(modelConfigs))

	for i := range modelConfigs {
		go func(modelConfig ModelConfig) {
			var owner *mgmtpb.User
			if modelConfig.OwnerType == string(resource.User) {
				resp, err := mgmtPrivateServiceClient.GetUserAdmin(context.Background(), &mgmtpb.GetUserAdminRequest{
					UserId: modelConfig.OwnerID,
				})
				if err != nil {
					logger.Fatal(err.Error())
				}
				owner = resp.GetUser()
			} else if modelConfig.OwnerType == string(resource.Organization) {
				resp, err := mgmtPrivateServiceClient.GetOrganizationAdmin(context.Background(), &mgmtpb.GetOrganizationAdminRequest{
					OrganizationId: modelConfig.OwnerID,
				})
				if err != nil {
					logger.Fatal(err.Error())
				}
				owner = resp.GetOrganization().GetOwner()
			}

			sCtx := gateway.InjectOwnerToContext(context.Background(), owner)

			defer wg.Done()
			configuration, err := structpb.NewStruct(modelConfig.Configuration)
			if err != nil {
				log.Fatal("structpb.NewValue: ", err)
				return
			}

			if _, err = modelPublicServiceClient.GetNamespaceModel(sCtx, &modelpb.GetNamespaceModelRequest{
				NamespaceId: modelConfig.OwnerID,
				ModelId:     modelConfig.ID,
				View:        modelpb.View_VIEW_FULL.Enum(),
			}); err != nil {
				logger.Info("Creating model: " + modelConfig.ID)

				model := &modelpb.Model{
					Id:              modelConfig.ID,
					Description:     &modelConfig.Description,
					ModelDefinition: modelConfig.ModelDefinition,
					Visibility:      modelpb.Model_VISIBILITY_PUBLIC,
					Task:            commonpb.Task(commonpb.Task_value[modelConfig.Task]),
					Region:          modelConfig.Region,
					Hardware:        modelConfig.Hardware,
					Configuration:   configuration,
				}
				if _, err = modelPublicServiceClient.CreateNamespaceModel(sCtx, &modelpb.CreateNamespaceModelRequest{
					NamespaceId: modelConfig.OwnerID,
					Model:       model,
				}); err != nil {
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
			if _, err = modelPrivateServiceClient.DeployNamespaceModelAdmin(context.Background(), &modelpb.DeployNamespaceModelAdminRequest{
				NamespaceId: modelConfig.OwnerID,
				ModelId:     modelConfig.ID,
				Version:     modelConfig.Version,
			}); err != nil {
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
		}(modelConfigs[i])
	}

	wg.Wait()

	logger.Info("creating and deploying models done!")
}

func newClients(logger *zap.Logger) (
	mgmtpb.MgmtPrivateServiceClient,
	modelpb.ModelPublicServiceClient,
	modelpb.ModelPrivateServiceClient,
	func(),
) {
	closeFuncs := map[string]func() error{}

	// Initialize mgmt private service client
	mgmtPrivateServiceClient, mclose, err := grpcclientx.NewMgmtPrivateClient(config.Config.MgmtBackend)
	if err != nil {
		logger.Fatal("failed to create mgmt private service client", zap.Error(err))
	}
	closeFuncs["mgmtPrivate"] = mclose

	// Initialize model public service client
	host := os.Getenv("MODEL_BACKEND_HOST")
	if host == "" {
		host = "localhost"
	}

	modelPublicServiceClient, mclose, err := grpcclientx.NewModelPublicClient(client.ServiceConfig{
		Host:        host,
		PrivatePort: config.Config.Server.PrivatePort,
		PublicPort:  config.Config.Server.PublicPort,
		HTTPS:       config.Config.Server.HTTPS,
	})
	if err != nil {
		logger.Fatal("failed to create model public service client", zap.Error(err))
	}
	closeFuncs["modelPublic"] = mclose

	// Initialize model private service client
	modelPrivateServiceClient, mclose, err := grpcclientx.NewModelPrivateClient(client.ServiceConfig{
		Host:        host,
		PrivatePort: config.Config.Server.PrivatePort,
		PublicPort:  config.Config.Server.PublicPort,
		HTTPS:       config.Config.Server.HTTPS,
	})
	if err != nil {
		logger.Fatal("failed to create model private service client", zap.Error(err))
	}
	closeFuncs["modelPrivate"] = mclose

	closer := func() {
		for conn, fn := range closeFuncs {
			if err := fn(); err != nil {
				logger.Error("Failed to close conn", zap.Error(err), zap.String("conn", conn))
			}
		}
	}

	return mgmtPrivateServiceClient, modelPublicServiceClient, modelPrivateServiceClient, closer
}
