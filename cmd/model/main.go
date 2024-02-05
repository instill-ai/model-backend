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

	"github.com/gogo/status"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/structpb"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"

	"github.com/instill-ai/model-backend/config"
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

	userID := fmt.Sprintf("users/%s", config.Config.InitModel.OwnerID)

	resp, err := mgmtPrivateServiceClient.GetUserAdmin(ctx, &mgmtPB.GetUserAdminRequest{
		Name: userID,
	})
	if err != nil {
		logger.Fatal(err.Error())
	}

	ctx = middleware.InjectOwnerToContext(ctx, resp.GetUser())

	var modelConfigs []ModelConfig
	err = utils.GetJSON(config.Config.InitModel.Path, &modelConfigs)
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
			getResp, err := modelPublicServiceClient.GetUserModel(ctx, &modelPB.GetUserModelRequest{
				Name: fmt.Sprintf("%s/models/%s", userID, modelConfig.ID),
				View: modelPB.View_VIEW_FULL.Enum(),
			})
			if err == nil {
				var existedModelConfig datamodel.GitHubModelConfiguration
				b, err := getResp.Model.Configuration.MarshalJSON()
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
					_, err = modelPublicServiceClient.DeleteUserModel(ctx, &modelPB.DeleteUserModelRequest{
						Name: fmt.Sprintf("%s/models/%s", userID, modelConfig.ID),
					})
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
			createOperation, err := modelPublicServiceClient.CreateUserModel(ctx, &modelPB.CreateUserModelRequest{
				Model: &modelPB.Model{
					Id:              modelConfig.ID,
					Description:     &modelConfig.Description,
					Task:            utils.Tasks[strings.ToUpper(modelConfig.Task)],
					ModelDefinition: modelConfig.ModelDefinition,
					Configuration:   configuration,
					Visibility:      modelPB.Model_VISIBILITY_PUBLIC,
				},
				Parent: userID,
			})
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
						Name: createOperation.Operation.Name,
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
					_, err := modelPublicServiceClient.DeployUserModel(ctx, &modelPB.DeployUserModelRequest{
						Name: fmt.Sprintf("%s/models/%s", userID, modelConfig.ID),
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
				}
				return
			}
		}(modelConfig)
	}

	wg.Wait()

	logger.Info("creating and deploying models done!")
	span.End()
}
