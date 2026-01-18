package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gogo/status"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"

	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/utils"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	mgmtpb "github.com/instill-ai/protogen-go/mgmt/v1beta"
	modelpb "github.com/instill-ai/protogen-go/model/v1alpha"
	clientx "github.com/instill-ai/x/client"
	clientgrpcx "github.com/instill-ai/x/client/grpc"
	logx "github.com/instill-ai/x/log"
	gatewayx "github.com/instill-ai/x/server/grpc/gateway"
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

	ctx, cancel := context.WithTimeout(context.Background(), 3600*time.Second)
	defer cancel()

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
	err := utils.GetJSON(config.Config.InitModel.Inventory, &modelConfigs)
	if err != nil {
		logger.Fatal(err.Error())
	}

	logger.Info("Creating models ...")

	var wg sync.WaitGroup
	wg.Add(len(modelConfigs))

	for i := range modelConfigs {
		go func(modelConfig ModelConfig) {
			// Get the owner UID by checking the namespace
			checkResp, err := mgmtPrivateServiceClient.CheckNamespaceAdmin(ctx, &mgmtpb.CheckNamespaceAdminRequest{
				Id: modelConfig.OwnerID,
			})
			if err != nil {
				logger.Fatal(err.Error())
			}
			ownerUID := checkResp.GetUid()

			sCtx := gatewayx.InjectOwnerToContext(ctx, ownerUID)

			defer wg.Done()
			configuration, err := structpb.NewStruct(modelConfig.Configuration)
			if err != nil {
				log.Fatal("structpb.NewValue: ", err)
				return
			}

			if _, err = modelPublicServiceClient.GetNamespaceModel(sCtx, &modelpb.GetNamespaceModelRequest{
				Name: fmt.Sprintf("namespaces/%s/models/%s", modelConfig.OwnerID, modelConfig.ID),
				View: modelpb.View_VIEW_FULL.Enum(),
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
					Parent: fmt.Sprintf("namespaces/%s", modelConfig.OwnerID),
					Model:  model,
				}); err != nil {
					logger.Fatal("Created model err: " + err.Error())
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
			if _, err = modelPrivateServiceClient.DeployNamespaceModelAdmin(ctx, &modelpb.DeployNamespaceModelAdminRequest{
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
	mgmtPrivateServiceClient, mgmtPrivateClose, err := clientgrpcx.NewClient[mgmtpb.MgmtPrivateServiceClient](
		clientgrpcx.WithServiceConfig(config.Config.MgmtBackend),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create mgmt private service client", zap.Error(err))
	}
	closeFuncs["mgmtPrivate"] = mgmtPrivateClose

	modelPublicServiceClient, modelPublicClose, err := clientgrpcx.NewClient[modelpb.ModelPublicServiceClient](
		clientgrpcx.WithServiceConfig(clientx.ServiceConfig{
			Host:        "model-backend", // running in Docker container only
			PrivatePort: config.Config.Server.PrivatePort,
			PublicPort:  config.Config.Server.PublicPort,
			HTTPS:       config.Config.Server.HTTPS,
		}),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create model public service client", zap.Error(err))
	}
	closeFuncs["modelPublic"] = modelPublicClose

	// Initialize model private service client
	modelPrivateServiceClient, modelPrivateClose, err := clientgrpcx.NewClient[modelpb.ModelPrivateServiceClient](
		clientgrpcx.WithServiceConfig(clientx.ServiceConfig{
			Host:        "model-backend", // running in Docker container only
			PrivatePort: config.Config.Server.PrivatePort,
			PublicPort:  config.Config.Server.PublicPort,
			HTTPS:       config.Config.Server.HTTPS,
		}),
		clientgrpcx.WithSetOTELClientHandler(config.Config.OTELCollector.Enable),
	)
	if err != nil {
		logger.Fatal("failed to create model private service client", zap.Error(err))
	}
	closeFuncs["modelPrivate"] = modelPrivateClose

	closer := func() {
		for conn, fn := range closeFuncs {
			if err := fn(); err != nil {
				logger.Error("Failed to close conn", zap.Error(err), zap.String("conn", conn))
			}
		}
	}

	return mgmtPrivateServiceClient, modelPublicServiceClient, modelPrivateServiceClient, closer
}
