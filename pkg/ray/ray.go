package ray

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v3"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/ray/rayserver"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type Ray interface {
	// grpc
	ModelReady(ctx context.Context, modelName string, version string) (*modelPB.State, string, error)
	ModelMetadataRequest(ctx context.Context, modelName string, version string) *rayserver.ModelMetadataResponse
	ModelInferRequest(ctx context.Context, task commonPB.Task, inferInput InferInput, modelName string, version string, modelMetadata *rayserver.ModelMetadataResponse) (*rayserver.RayServiceCallResponse, error)

	// standard
	IsRayServerReady(ctx context.Context) bool
	UpdateContainerizedModel(ctx context.Context, modelName string, userID string, imageName string, version string, hardware string, isDeploy bool) error
	Init()
	Close()
}

type ray struct {
	rayClient      rayserver.RayServiceClient
	rayServeClient rayserver.RayServeAPIServiceClient
	rayHTTPClient  *http.Client
	connection     *grpc.ClientConn
	configFilePath string
	configChan     chan ApplicationWithAction
	doneChan       chan error
}

func NewRay() Ray {
	rayService := &ray{}
	rayService.Init()
	return rayService
}

func (r *ray) Init() {
	// Connect to gRPC server
	conn, err := grpc.Dial(
		config.Config.RayServer.GrpcURI,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.Config.Server.MaxDataSize*constant.MB),
			grpc.MaxCallSendMsgSize(config.Config.Server.MaxDataSize*constant.MB),
		),
	)

	if err != nil {
		log.Fatalf("Couldn't connect to endpoint %s: %v", config.Config.RayServer.GrpcURI, err)
	}

	// Create client from gRPC server connection
	r.connection = conn
	r.rayClient = rayserver.NewRayServiceClient(conn)
	r.rayServeClient = rayserver.NewRayServeAPIServiceClient(conn)
	r.rayHTTPClient = &http.Client{Timeout: 60 * time.Second}
	r.configChan = make(chan ApplicationWithAction, 10000)
	r.doneChan = make(chan error, 10000)
	r.configFilePath = path.Join(config.Config.RayServer.ModelStore, "deploy.yaml")

	var modelDeploymentConfig ModelDeploymentConfig
	isCorrupted := false
	currentConfigFile, err := os.ReadFile(r.configFilePath)
	if err != nil {
		isCorrupted = true
	}
	err = yaml.Unmarshal(currentConfigFile, &modelDeploymentConfig)
	if err != nil {
		isCorrupted = true
	}

	if _, err := os.Stat(r.configFilePath); os.IsNotExist(err) || isCorrupted {
		initDeployConfig := ModelDeploymentConfig{
			Applications: []Application{},
		}
		initConfigData, err := yaml.Marshal(&initDeployConfig)
		if err != nil {
			fmt.Printf("error while Marshaling deployment config: %v", err)
		}
		if err := os.WriteFile(r.configFilePath, initConfigData, 0666); err != nil {
			fmt.Printf("error creating deployment config: %v", err)
		}
	}

	// avoid race condition with file writing
	// add/remove application entries
	go r.sync()
}

func (r *ray) IsRayServerReady(ctx context.Context) bool {
	logger, _ := custom_logger.GetZapLogger(ctx)

	resp, err := r.rayServeClient.Healthz(ctx, &rayserver.HealthzRequest{})
	if err != nil {
		logger.Error(err.Error())
		return false
	}

	if resp != nil && resp.Message == "success" {
		return true
	} else {
		return false
	}
}

func (r *ray) ModelReady(ctx context.Context, modelName string, version string) (*modelPB.State, string, error) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName, version)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.Replace(fmt.Sprintf("http://%s/api/serve/applications/", config.Config.RayServer.GrpcURI), "9000", "8265", 1), http.NoBody)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", err
	}
	resp, err := r.rayHTTPClient.Do(req)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", err
	}
	defer resp.Body.Close()

	var applicationStatus rayserver.GetApplicationStatus
	err = json.NewDecoder(resp.Body).Decode(&applicationStatus)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", err
	}

	application, ok := applicationStatus.Applications[applicationMetadatValue]
	if !ok {
		return modelPB.State_STATE_ERROR.Enum(), "", status.New(codes.NotFound, "Model version not found").Err()
	}

	switch application.Status {
	case rayserver.ApplicationStatusStrUnhealthy, rayserver.ApplicationStatusStrRunning:
		for i := range application.Deployments {
			switch application.Deployments[i].Status {
			case rayserver.DeploymentStatusStrHealthy:
				return modelPB.State_STATE_ACTIVE.Enum(), application.Deployments[i].Message, nil
			case rayserver.DeploymentStatusStrUpdating:
				return modelPB.State_STATE_UNSPECIFIED.Enum(), application.Deployments[i].Message, nil
			case rayserver.DeploymentStatusStrUpscaling, rayserver.DeploymentStatusStrDownscaling:
				return modelPB.State_STATE_SCALING.Enum(), application.Deployments[i].Message, nil
			case rayserver.DeploymentStatusStrUnhealthy:
				return modelPB.State_STATE_ERROR.Enum(), application.Deployments[i].Message, nil
			}
		}
		return modelPB.State_STATE_ERROR.Enum(), application.Message, nil
	case rayserver.ApplicationStatusStrDeploying, rayserver.ApplicationStatusStrDeleting:
		return modelPB.State_STATE_UNSPECIFIED.Enum(), application.Message, nil
	case rayserver.ApplicationStatusStrNotStarted:
		return modelPB.State_STATE_OFFLINE.Enum(), application.Message, nil
	case rayserver.ApplicationStatusStrDeployFailed:
		return modelPB.State_STATE_ERROR.Enum(), application.Message, nil
	}

	return modelPB.State_STATE_UNSPECIFIED.Enum(), application.Message, nil
}

func (r *ray) ModelMetadataRequest(ctx context.Context, modelName string, version string) *rayserver.ModelMetadataResponse {
	logger, _ := custom_logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName, version)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "application", applicationMetadatValue)

	// Create status request for a given model
	modelMetadataRequest := rayserver.ModelMetadataRequest{
		Name:    modelName,
		Version: version,
	}
	// Submit modelMetadata request to server
	modelMetadataResponse, err := r.rayClient.ModelMetadata(ctx, &modelMetadataRequest)
	if err != nil {
		log.Printf("Couldn't get server model metadata: %v", err)
	}
	return modelMetadataResponse
}

func (r *ray) ModelInferRequest(ctx context.Context, task commonPB.Task, inferInput InferInput, modelName string, version string, modelMetadata *rayserver.ModelMetadataResponse) (*rayserver.RayServiceCallResponse, error) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName, version)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "application", applicationMetadatValue)

	modelInferRequest := PreProcess(modelName, version, inferInput, task, modelMetadata)

	modelInferResponse, err := r.rayClient.XCall__(ctx, modelInferRequest)
	if err != nil {
		logger.Error(fmt.Sprintf("Error processing InferRequest: %s", err.Error()))
		return &rayserver.RayServiceCallResponse{}, err
	}

	return modelInferResponse, nil
}

func (r *ray) UpdateContainerizedModel(ctx context.Context, modelName string, userID string, imageName string, version string, hardware string, isDeploy bool) error {

	logger, _ := custom_logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName, version)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	runOptions := []string{
		"--tls-verify=false",
		"--pull=always",
		"--rm",
		"-v /home/ray/ray_pb2.py:/home/ray/ray_pb2.py",
		"-v /home/ray/ray_pb2.pyi:/home/ray/ray_pb2.pyi",
		"-v /home/ray/ray_pb2_grpc.py:/home/ray/ray_pb2_grpc.py",
	}

	if isDeploy {
		accelerator, ok := SupportedAcceleratorType[hardware]
		if !ok {
			logger.Warn("accelerator type(hardware) not supported, setting it as custom resource")
			runOptions = append(runOptions,
				fmt.Sprintf("-e %s=%v", EnvTotalVRAM, config.Config.RayServer.Vram),
				fmt.Sprintf("-e %s=%v", EnvNumOfGPUs, 1),
				fmt.Sprintf("-e %s=%s", EnvRayCustomResource, hardware),
				"--device nvidia.com/gpu=all",
			)
		} else {
			if accelerator == SupportedAcceleratorType["CPU"] {
				runOptions = append(runOptions, fmt.Sprintf("-e %s=%v", EnvNumOfCPUs, 2))
			} else if accelerator == SupportedAcceleratorType["GPU"] {
				runOptions = append(runOptions,
					fmt.Sprintf("-e %s=%v", EnvTotalVRAM, config.Config.RayServer.Vram),
					fmt.Sprintf("-e %s=%v", EnvNumOfGPUs, 1),
					"--device nvidia.com/gpu=all",
				)
			} else {
				runOptions = append(runOptions,
					fmt.Sprintf("-e %s=%s", EnvRayAcceleratorType, accelerator),
					fmt.Sprintf("-e %s=%v", EnvTotalVRAM, SupportedAcceleratorTypeMemory[hardware]),
					"--device nvidia.com/gpu=all",
				)
			}
		}
	}

	// TODO: Support custom resource configs for deployment in the future
	if userID == "instill-ai" {
		runOptions = append(runOptions,
			fmt.Sprintf("-e %s=%v", EnvNumOfMinReplicas, 1),
			fmt.Sprintf("-e %s=%v", EnvNumOfMaxReplicas, 1),
		)
	} else {
		runOptions = append(runOptions,
			fmt.Sprintf("-e %s=%v", EnvNumOfMinReplicas, 0),
			fmt.Sprintf("-e %s=%v", EnvNumOfMaxReplicas, 1),
		)
	}

	applicationConfig := Application{
		Name:        applicationMetadatValue,
		ImportPath:  "model:entrypoint",
		RoutePrefix: "/" + applicationMetadatValue,
		RuntimeEnv: RuntimeEnv{
			Container: Container{
				Image:      fmt.Sprintf("%s:%v/%s/%s:%s", config.Config.Registry.Host, config.Config.Registry.Port, userID, imageName, version),
				RunOptions: runOptions,
			},
		},
	}

	r.configChan <- ApplicationWithAction{
		Application: applicationConfig,
		IsDeploy:    isDeploy,
	}

	return <-r.doneChan
}

func (r *ray) sync() {
	for {
		applicationWithAction := <-r.configChan

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		logger, _ := custom_logger.GetZapLogger(ctx)
		var modelDeploymentConfig ModelDeploymentConfig

		currentConfigFile, err := os.ReadFile(r.configFilePath)
		if err != nil {
			logger.Error(fmt.Sprintf("error while reading deployment config: %v", err))
		}
		err = yaml.Unmarshal(currentConfigFile, &modelDeploymentConfig)
		if err != nil {
			logger.Error(fmt.Sprintf("error while Unmarshaling deployment config: %v", err))
		}

		newApplications := []Application{}
		switch applicationWithAction.IsDeploy {
		case true:
			for _, app := range modelDeploymentConfig.Applications {
				if app.Name != applicationWithAction.Application.Name {
					newApplications = append(newApplications, app)
				}
			}
			modelDeploymentConfig.Applications = newApplications
			modelDeploymentConfig.Applications = append(modelDeploymentConfig.Applications, applicationWithAction.Application)
		case false:
			for _, app := range modelDeploymentConfig.Applications {
				if app.Name != applicationWithAction.Application.Name {
					newApplications = append(newApplications, app)
				}
			}
			modelDeploymentConfig.Applications = newApplications
		}

		modelDeploymentConfigData, err := yaml.Marshal(modelDeploymentConfig)
		if err != nil {
			logger.Error(fmt.Sprintf("error while Marshaling YAML deployment config: %v", err))
		}

		if err := os.WriteFile(r.configFilePath, modelDeploymentConfigData, 0666); err != nil {
			logger.Error(fmt.Sprintf("error creating deployment config: %v", err))
		}

		modelDeploymentConfigJSON, err := json.Marshal(modelDeploymentConfig)
		if err != nil {
			logger.Error(fmt.Sprintf("error while Marshaling JSON deployment config: %v", err))
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, strings.ReplaceAll(fmt.Sprintf("http://%s/api/serve/applications/", config.Config.RayServer.GrpcURI), "9000", "8265"), bytes.NewBuffer(modelDeploymentConfigJSON))
		if err != nil {
			logger.Error(fmt.Sprintf("error while creating deployment request: %v", err))
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := r.rayHTTPClient.Do(req)
		if err != nil {
			logger.Error(fmt.Sprintf("error while sending deployment request: %v", err))
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			logger.Error(err.Error())
		}
		if resp.StatusCode != http.StatusOK {
			err = fmt.Errorf("error while sending deployment request, status code: %v, description: %v", resp.StatusCode, string(bodyBytes))
		}

		resp.Body.Close()
		cancel()

		r.doneChan <- err
	}
}

func (r *ray) Close() {
	if r.connection != nil {
		r.connection.Close()
	}
}
