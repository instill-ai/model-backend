package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/gogo/status"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	custom_otel "github.com/instill-ai/model-backend/pkg/logger/otel"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/util"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

type ModelConfig struct {
	ID              string                 `json:"id"`
	Description     string                 `json:"description"`
	Task            string                 `json:"task"`
	ModelDefinition string                 `json:"model_definition"`
	Configuration   map[string]interface{} `json:"configuration"`
}

// InitModelPublicServiceClient initialises a ModelServiceClient instance
func InitModelPublicServiceClient(ctx context.Context) (modelPB.ModelPublicServiceClient, *grpc.ClientConn) {
	logger, _ := logger.GetZapLogger(ctx)

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

	// setup tracing and metrics
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if tp, err := custom_otel.SetupTracing(ctx, "model-backend"); err != nil {
		panic(err)
	} else {
		defer func() {
			err = tp.Shutdown(ctx)
		}()
	}

	if mp, err := custom_otel.SetupMetrics(ctx, "model-backend"); err != nil {
		panic(err)
	} else {
		defer func() {
			err = mp.Shutdown(ctx)
		}()
	}

	ctx, span := otel.Tracer("main-tracer").Start(ctx,
		"main",
	)

	logger, _ := logger.GetZapLogger(ctx)

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

	modelPublicServiceClient, modelPublicServiceClientConn := InitModelPublicServiceClient(ctx)
	if modelPublicServiceClientConn != nil {
		defer modelPublicServiceClientConn.Close()
	}

	var modelConfigs []ModelConfig
	err := util.GetJSON(config.Config.InitModel.Path, &modelConfigs)
	if err != nil {
		logger.Fatal(err.Error())
	}
	logger.Info("Creating models ...")
	for _, modelConfig := range modelConfigs {
		configuration, err := structpb.NewStruct(modelConfig.Configuration)
		if err != nil {
			log.Fatal("structpb.NewValue: ", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()
		logger.Info("Creating model: " + modelConfig.ID)
		createOperation, err := modelPublicServiceClient.CreateModel(ctx, &modelPB.CreateModelRequest{
			Model: &modelPB.Model{
				Id:              modelConfig.ID,
				Description:     &modelConfig.Description,
				Task:            modelPB.Model_Task(util.Tasks[strings.ToUpper(modelConfig.Task)]),
				ModelDefinition: modelConfig.ModelDefinition,
				Configuration:   configuration,
				Visibility:      modelPB.Model_VISIBILITY_PUBLIC,
			},
		})
		if err != nil {
			logger.Info(fmt.Sprintf("Created model err: %v", err))
			if e, ok := status.FromError(err); ok {
				if e.Code() != codes.AlreadyExists {
					logger.Fatal("handler.CreateModel: " + err.Error())
				}
			}
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
				}
				isCreated = operation.Operation.Done
				time.Sleep(1 * time.Second)
			}
			if !isCreated {
				logger.Fatal("handler.CreateModel: " + err.Error())
			}
		}

		time.Sleep(5 * time.Second) // make sure controller updated the state.
		ctx, cancel = context.WithTimeout(context.Background(), 6000*time.Second)
		defer cancel()
		logger.Info("Deploying model: " + modelConfig.ID)
		deployOperation, err := modelPublicServiceClient.DeployModel(ctx, &modelPB.DeployModelRequest{
			Name: fmt.Sprintf("models/%s", modelConfig.ID),
		})
		if err != nil {
			logger.Info(fmt.Sprintf("Deployed model err: %v", err))
			if e, ok := status.FromError(err); ok {
				if e.Code() == codes.FailedPrecondition {
					continue
				}
			}
			logger.Fatal("handler.DeployModel: " + err.Error())
		}

		isDeployed := false
		startTime := time.Now()
		for {
			if isDeployed || time.Since(startTime) > 150*time.Minute {
				break
			}
			operation, err := modelPublicServiceClient.GetModelOperation(ctx, &modelPB.GetModelOperationRequest{
				Name: deployOperation.Operation.Name,
			})
			if err != nil {
				logger.Fatal("handler.GetModelOperation: " + err.Error())
			}
			isDeployed = operation.Operation.Done
			time.Sleep(5 * time.Second)
		}
	}
	logger.Info("Creating models done!")
	span.End()
}
