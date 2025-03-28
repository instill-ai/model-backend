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
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"gopkg.in/yaml.v3"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/ray/rayserver"
	"github.com/redis/go-redis/v9"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
)

type Ray interface {
	// grpc
	ModelReady(ctx context.Context, modelName string, version string) (*modelpb.State, string, int, error)
	ModelInferRequest(ctx context.Context, task commonpb.Task, req *modelpb.TriggerNamespaceModelRequest, modelName string, version string) (*rayserver.CallResponse, error)

	// standard
	IsRayServerReady(ctx context.Context) bool
	UpdateContainerizedModel(ctx context.Context, modelName string, userID string, imageName string, version string, hardware string, action Action, scalingConfig []string, numOfGPU string) error
	Init(rc *redis.Client)
	Close()
}

type ray struct {
	rayClient      rayserver.RayServiceClient
	rayServeClient rayserver.RayServeAPIServiceClient
	rayHTTPClient  *http.Client
	redisClient    *redis.Client
	connection     *grpc.ClientConn
	configFilePath string
	configChan     chan ApplicationWithAction
	doneChan       chan error
}

var once sync.Once
var rayService *ray

func NewRay(rc *redis.Client) Ray {
	once.Do(func() {
		rayService = &ray{}
		rayService.Init(rc)
	})
	return rayService
}

func (r *ray) Init(rc *redis.Client) {
	ctx := context.Background()
	// Connect to gRPC server
	conn, err := grpc.NewClient(
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

	r.redisClient = rc

	// Create client from gRPC server connection
	r.connection = conn
	r.rayClient = rayserver.NewRayServiceClient(conn)
	r.rayServeClient = rayserver.NewRayServeAPIServiceClient(conn)
	r.rayHTTPClient = &http.Client{Timeout: time.Minute}
	r.configChan = make(chan ApplicationWithAction, 10000)
	r.doneChan = make(chan error, 10000)
	r.configFilePath = path.Join(config.Config.RayServer.ModelStore, "deploy.yaml")

	isCorrupted := false
	currentConfigFile, err := os.ReadFile(r.configFilePath)
	if err != nil {
		isCorrupted = true
	}

	if _, err := os.Stat(r.configFilePath); !os.IsNotExist(err) && !isCorrupted {
		r.redisClient.Set(
			ctx,
			RayDeploymentKey,
			currentConfigFile,
			0,
		)
	}

	// avoid race condition with file writing
	// add/remove application entries
	go r.sync()

	// sync potential missing applications
	if err = r.UpdateContainerizedModel(context.Background(), "", "", "", "", "", Sync, []string{}, "1"); err != nil {
		fmt.Printf("error syncing deployment config: %v\n", err)
	}
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

func (r *ray) ModelReady(ctx context.Context, modelName string, version string) (*modelpb.State, string, int, error) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName, version)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.Replace(fmt.Sprintf("http://%s/api/serve/applications/", config.Config.RayServer.GrpcURI), "9000", "8265", 1), http.NoBody)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", 0, err
	}
	resp, err := r.rayHTTPClient.Do(req)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", 0, err
	}
	defer resp.Body.Close()

	var applicationStatus rayserver.GetApplicationStatus
	err = json.NewDecoder(resp.Body).Decode(&applicationStatus)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", 0, err
	}

	application, ok := applicationStatus.Applications[applicationMetadatValue]
	if !ok {
		return modelpb.State_STATE_OFFLINE.Enum(), "", 0, nil
	}

	switch application.Status {
	case rayserver.ApplicationStatusStrUnhealthy, rayserver.ApplicationStatusStrRunning:
		for i := range application.Deployments {
			numOfReplicas := len(application.Deployments[i].Replicas)
			switch application.Deployments[i].Status {
			case rayserver.DeploymentStatusStrHealthy:
				if numOfReplicas == 0 {
					return modelpb.State_STATE_OFFLINE.Enum(), application.Deployments[i].Message, numOfReplicas, nil
				} else {
					return modelpb.State_STATE_ACTIVE.Enum(), application.Deployments[i].Message, numOfReplicas, nil
				}
			case rayserver.DeploymentStatusStrUpdating:
				return modelpb.State_STATE_STARTING.Enum(), application.Deployments[i].Message, numOfReplicas, nil
			case rayserver.DeploymentStatusStrUpscaling:
				return modelpb.State_STATE_SCALING_UP.Enum(), application.Deployments[i].Message, numOfReplicas, nil
			case rayserver.DeploymentStatusStrDownscaling:
				return modelpb.State_STATE_SCALING_DOWN.Enum(), application.Deployments[i].Message, numOfReplicas, nil
			case rayserver.DeploymentStatusStrUnhealthy:
				return modelpb.State_STATE_ERROR.Enum(), application.Deployments[i].Message, 0, nil
			}
		}
		return modelpb.State_STATE_ERROR.Enum(), application.Message, 0, nil
	case rayserver.ApplicationStatusStrDeploying:
		for i := range application.Deployments {
			switch application.Deployments[i].Status {
			case rayserver.DeploymentStatusStrUpdating:
				return modelpb.State_STATE_SCALING_UP.Enum(), application.Deployments[i].Message, 0, nil
			case rayserver.DeploymentStatusStrUnhealthy:
				return modelpb.State_STATE_ERROR.Enum(), application.Deployments[i].Message, 0, nil
			}
		}
		return modelpb.State_STATE_STARTING.Enum(), application.Message, 0, nil
	case rayserver.ApplicationStatusStrDeleting:
		for i := range application.Deployments {
			switch application.Deployments[i].Status {
			case rayserver.DeploymentStatusStrUpdating:
				return modelpb.State_STATE_SCALING_DOWN.Enum(), application.Deployments[i].Message, 0, nil
			case rayserver.DeploymentStatusStrUnhealthy:
				return modelpb.State_STATE_ERROR.Enum(), application.Deployments[i].Message, 0, nil
			}
		}
		return modelpb.State_STATE_STARTING.Enum(), application.Message, 0, nil
	case rayserver.ApplicationStatusStrNotStarted:
		return modelpb.State_STATE_OFFLINE.Enum(), application.Message, 0, nil
	case rayserver.ApplicationStatusStrDeployFailed:
		return modelpb.State_STATE_ERROR.Enum(), application.Message, 0, nil
	}

	return modelpb.State_STATE_ERROR.Enum(), application.Message, 0, nil
}

func (r *ray) ModelInferRequest(ctx context.Context, task commonpb.Task, req *modelpb.TriggerNamespaceModelRequest, modelName string, version string) (*rayserver.CallResponse, error) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName, version)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "application", applicationMetadatValue)

	rayTriggerReq := &rayserver.CallRequest{
		TaskInputs: req.GetTaskInputs(),
	}

	modelInferResponse, err := r.rayClient.XCall__(ctx, rayTriggerReq)
	if err != nil {
		logger.Error(fmt.Sprintf("Error processing InferRequest: %s", err.Error()))
		return &rayserver.CallResponse{}, err
	}

	return modelInferResponse, nil
}

func (r *ray) UpdateContainerizedModel(ctx context.Context, modelName string, userID string, imageName string, version string, hardware string, action Action, scalingConfig []string, numOfGPU string) error {
	logger, _ := custom_logger.GetZapLogger(ctx)

	var err error
	applicationMetadatValue := ""
	runOptions := []string{}

	if action != Sync {
		applicationMetadatValue, err = GetApplicationMetadaValue(modelName, version)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	if action == Deploy {
		runOptions = append(runOptions,
			"--tls-verify=false",
			"--pull=always",
			"--rm",
			"-v /home/ray/ray_pb2.py:/home/ray/ray_pb2.py",
			"-v /home/ray/ray_pb2.pyi:/home/ray/ray_pb2.pyi",
			"-v /home/ray/ray_pb2_grpc.py:/home/ray/ray_pb2_grpc.py")
		runOptions = append(runOptions, r.setHardwareRunOptions(hardware, numOfGPU)...)
		if len(scalingConfig) > 0 {
			runOptions = append(runOptions, scalingConfig...)
		} else {
			runOptions = append(runOptions,
				fmt.Sprintf("-e %s=%v", EnvNumOfMinReplicas, 0),
				fmt.Sprintf("-e %s=%v", EnvNumOfMaxReplicas, 10),
			)
		}
	}

	applicationConfig := Application{
		Name:        applicationMetadatValue,
		ImportPath:  "_model:entrypoint",
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
		Action:      action,
	}

	return <-r.doneChan
}

func (r *ray) setHardwareRunOptions(hardware string, numOfGPU string) []string {
	logger, _ := custom_logger.GetZapLogger(context.Background())
	runOptions := []string{}

	accelerator, ok := SupportedAcceleratorType[hardware]
	if !ok {
		logger.Warn("accelerator type(hardware) not supported, setting it as custom resource")
		return append(runOptions,
			fmt.Sprintf("-e %s=%v", EnvTotalVRAM, config.Config.RayServer.Vram),
			fmt.Sprintf("-e %s=%v", EnvNumOfGPUs, numOfGPU),
			fmt.Sprintf("-e %s=%s", EnvRayCustomResource, hardware),
			"--device nvidia.com/gpu=all",
		)
	}

	switch accelerator {
	case SupportedAcceleratorType["CPU"]:
		runOptions = append(runOptions, fmt.Sprintf("-e %s=%v", EnvNumOfCPUs, 1))
	case SupportedAcceleratorType["GPU"]:
		runOptions = append(runOptions,
			fmt.Sprintf("-e %s=%v", EnvTotalVRAM, config.Config.RayServer.Vram),
			fmt.Sprintf("-e %s=%v", EnvNumOfGPUs, 1),
			"--device nvidia.com/gpu=all",
		)
	default:
		numOfGPUFloat, err := strconv.ParseFloat(numOfGPU, 64)
		if err != nil {
			numOfGPUFloat = 0
		}
		if numOfGPUFloat > 0 {
			runOptions = append(runOptions,
				fmt.Sprintf("-e %s=%s", EnvRayCustomResource, hardware),
				fmt.Sprintf("-e %s=%v", EnvNumOfGPUs, numOfGPU),
				"--device nvidia.com/gpu=all",
			)
		} else {
			runOptions = append(runOptions,
				fmt.Sprintf("-e %s=%s", EnvRayCustomResource, hardware),
				fmt.Sprintf("-e %s=%v", EnvTotalVRAM, SupportedAcceleratorTypeMemory[hardware]),
				"--device nvidia.com/gpu=all",
			)
		}
	}

	return runOptions
}

func (r *ray) sync() {
	for {
		applicationWithAction := <-r.configChan

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

		logger, _ := custom_logger.GetZapLogger(ctx)

		if applicationWithAction.Action == UpScale {
			// this is a pseudo trigger request to invoke model upscale
			// we do not care about the trigger result
			go func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.ReplaceAll(fmt.Sprintf("http://%s%s", config.Config.RayServer.GrpcURI, applicationWithAction.Application.RoutePrefix), "9000", "8000"), http.NoBody)
				if err != nil {
					logger.Error(fmt.Sprintf("error while creating upscale request: %v", err))
					return
				}
				resp, err := r.rayHTTPClient.Do(req)
				if err != nil {
					logger.Error(fmt.Sprintf("error while sending upscale request: %v", err))
					return
				}
				resp.Body.Close()
			}()

			cancel()
			r.doneChan <- nil
			continue
		}

		var modelDeploymentConfig ModelDeploymentConfig

		currentConfigFile, err := r.redisClient.Get(
			ctx,
			RayDeploymentKey,
		).Bytes()
		if err != nil {
			logger.Error(fmt.Sprintf("error while reading deployment config: %v", err))
		}
		err = yaml.Unmarshal(currentConfigFile, &modelDeploymentConfig)
		if err != nil {
			logger.Error(fmt.Sprintf("error while Unmarshaling deployment config: %v", err))
		}

		newApplications := []Application{}
		switch applicationWithAction.Action {
		case Deploy:
			for _, app := range modelDeploymentConfig.Applications {
				if app.Name != applicationWithAction.Application.Name {
					newApplications = append(newApplications, app)
				}
			}
			modelDeploymentConfig.Applications = newApplications
			modelDeploymentConfig.Applications = append(modelDeploymentConfig.Applications, applicationWithAction.Application)
		case Undeploy:
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

		if err := r.redisClient.Set(
			ctx,
			RayDeploymentKey,
			modelDeploymentConfigData,
			0,
		).Err(); err != nil {
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
	ctx := context.Background()

	logger, _ := custom_logger.GetZapLogger(ctx)

	currentConfigFile, err := r.redisClient.Get(
		ctx,
		RayDeploymentKey,
	).Bytes()
	if err != nil {
		logger.Error(fmt.Sprintf("error while reading deployment config: %v", err))
	}

	if err := os.WriteFile(r.configFilePath, currentConfigFile, 0666); err != nil {
		logger.Error(fmt.Sprintf("error creating deployment config: %v", err))
	}

	if r.connection != nil {
		r.connection.Close()
	}
	close(r.configChan)
	close(r.doneChan)
}
