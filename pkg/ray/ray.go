package ray

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"gopkg.in/yaml.v3"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/x/client"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
	raypb "github.com/instill-ai/protogen-go/model/ray"
	rayuserdefinedpb "github.com/instill-ai/protogen-go/model/ray/v1alpha"
	logx "github.com/instill-ai/x/log"
)

// Ray is the interface for the Ray service.
type Ray interface {
	// grpc
	ModelReady(ctx context.Context, modelName string, version string) (*modelpb.State, string, int, error)
	ModelInferRequest(ctx context.Context, task commonpb.Task, req *modelpb.TriggerNamespaceModelRequest, modelName string, version string) (*rayuserdefinedpb.CallResponse, error)

	// standard
	IsRayReady(ctx context.Context) bool
	UpdateContainerizedModel(ctx context.Context, modelName string, userID string, imageName string, version string, hardware string, action Action, numOfGPU string) error
	Init(rc *redis.Client)
	Close() error
}

type ray struct {
	userDefinedClient rayuserdefinedpb.UserDefinedServiceClient
	grpcClient        raypb.RayServeAPIServiceClient
	httpClient        *http.Client
	redisClient       *redis.Client
	connection        *grpc.ClientConn
	configFilePath    string
	configChan        chan ApplicationWithAction
	doneChan          chan error
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
	logger, _ := logx.GetZapLogger(ctx)

	// Connect to gRPC server
	conn, err := grpc.NewClient(
		fmt.Sprintf("%s:%d", config.Config.Ray.Host, config.Config.Ray.Port.GRPC),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(client.MaxPayloadSize),
			grpc.MaxCallSendMsgSize(client.MaxPayloadSize),
		),
	)

	if err != nil {
		logger.Fatal(fmt.Sprintf("Couldn't connect to gRPC endpoint %s: %v", fmt.Sprintf("%s:%d", config.Config.Ray.Host, config.Config.Ray.Port.GRPC), err))
	}

	r.redisClient = rc

	// Create client from gRPC server connection
	r.connection = conn
	r.userDefinedClient = rayuserdefinedpb.NewUserDefinedServiceClient(conn)
	r.grpcClient = raypb.NewRayServeAPIServiceClient(conn)
	r.httpClient = &http.Client{Timeout: time.Minute}
	r.configChan = make(chan ApplicationWithAction, 10000)
	r.doneChan = make(chan error, 10000)
	r.configFilePath = path.Join("/tmp", "deploy.yaml")

	if currentConfigFile, err := r.redisClient.Get(
		ctx,
		RayDeploymentKey,
	).Bytes(); err != nil {
		if configFile, err := os.ReadFile(r.configFilePath); err == nil {
			r.redisClient.Set(
				ctx,
				RayDeploymentKey,
				configFile,
				0,
			)
		}
	} else {
		if err := os.WriteFile(r.configFilePath, currentConfigFile, 0666); err != nil {
			logger.Error(fmt.Sprintf("error creating deployment config: %v", err))
		}
	}

	// avoid race condition with file writing
	// add/remove application entries
	go r.sync()

	// sync potential missing applications
	if err = r.UpdateContainerizedModel(context.Background(), "", "", "", "", "", Sync, "1"); err != nil {
		logger.Error(fmt.Sprintf("error syncing deployment config: %v", err))
	}
}

func (r *ray) IsRayReady(ctx context.Context) bool {
	logger, _ := logx.GetZapLogger(ctx)

	resp, err := r.grpcClient.Healthz(ctx, &raypb.HealthzRequest{})
	if err != nil {
		logger.Error(err.Error())
		return false
	}

	if resp != nil && resp.Message == "success" {
		return true
	}

	return false
}

// ModelReady returns the state of the model
func (r *ray) ModelReady(ctx context.Context, modelName string, version string) (*modelpb.State, string, int, error) {
	logger, _ := logx.GetZapLogger(ctx)

	applicationMetadataValue, err := GetApplicationMetadataValue(modelName, version)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s:%d/api/serve/applications/", config.Config.Ray.Host, config.Config.Ray.Port.DASHBOARD), http.NoBody)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", 0, err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", 0, err
	}
	defer resp.Body.Close()

	var applicationStatus GetApplicationStatus
	err = json.NewDecoder(resp.Body).Decode(&applicationStatus)
	if err != nil {
		logger.Error(err.Error())
		return nil, "", 0, err
	}

	application, ok := applicationStatus.Applications[applicationMetadataValue]
	if !ok {
		return modelpb.State_STATE_OFFLINE.Enum(), "", 0, nil
	}

	switch application.Status {
	case ApplicationStatusStrUnhealthy, ApplicationStatusStrRunning:
		for i := range application.Deployments {
			numOfReplicas := len(application.Deployments[i].Replicas)
			switch application.Deployments[i].Status {
			case DeploymentStatusStrHealthy:
				if numOfReplicas == 0 {
					return modelpb.State_STATE_OFFLINE.Enum(), application.Deployments[i].Message, numOfReplicas, nil
				} else {
					return modelpb.State_STATE_ACTIVE.Enum(), application.Deployments[i].Message, numOfReplicas, nil
				}
			case DeploymentStatusStrUpdating:
				return modelpb.State_STATE_STARTING.Enum(), application.Deployments[i].Message, numOfReplicas, nil
			case DeploymentStatusStrUpscaling:
				return modelpb.State_STATE_SCALING_UP.Enum(), application.Deployments[i].Message, numOfReplicas, nil
			case DeploymentStatusStrDownscaling:
				return modelpb.State_STATE_SCALING_DOWN.Enum(), application.Deployments[i].Message, numOfReplicas, nil
			case DeploymentStatusStrUnhealthy:
				return modelpb.State_STATE_ERROR.Enum(), application.Deployments[i].Message, 0, nil
			}
		}
		return modelpb.State_STATE_ERROR.Enum(), application.Message, 0, nil
	case ApplicationStatusStrDeploying:
		for i := range application.Deployments {
			switch application.Deployments[i].Status {
			case DeploymentStatusStrUpdating:
				return modelpb.State_STATE_SCALING_UP.Enum(), application.Deployments[i].Message, 0, nil
			case DeploymentStatusStrUnhealthy:
				return modelpb.State_STATE_ERROR.Enum(), application.Deployments[i].Message, 0, nil
			}
		}
		return modelpb.State_STATE_STARTING.Enum(), application.Message, 0, nil
	case ApplicationStatusStrDeleting:
		for i := range application.Deployments {
			switch application.Deployments[i].Status {
			case DeploymentStatusStrUpdating:
				return modelpb.State_STATE_SCALING_DOWN.Enum(), application.Deployments[i].Message, 0, nil
			case DeploymentStatusStrUnhealthy:
				return modelpb.State_STATE_ERROR.Enum(), application.Deployments[i].Message, 0, nil
			}
		}
		return modelpb.State_STATE_STARTING.Enum(), application.Message, 0, nil
	case ApplicationStatusStrNotStarted:
		return modelpb.State_STATE_OFFLINE.Enum(), application.Message, 0, nil
	case ApplicationStatusStrDeployFailed:
		return modelpb.State_STATE_ERROR.Enum(), application.Message, 0, nil
	}

	return modelpb.State_STATE_ERROR.Enum(), application.Message, 0, nil
}

func (r *ray) ModelInferRequest(ctx context.Context, task commonpb.Task, req *modelpb.TriggerNamespaceModelRequest, modelName string, version string) (*rayuserdefinedpb.CallResponse, error) {
	logger, _ := logx.GetZapLogger(ctx)

	applicationMetadataValue, err := GetApplicationMetadataValue(modelName, version)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "application", applicationMetadataValue)

	rayTriggerReq := &rayuserdefinedpb.CallRequest{
		TaskInputs: req.GetTaskInputs(),
	}

	modelInferResponse, err := r.userDefinedClient.XCall__(ctx, rayTriggerReq)
	if err != nil {
		logger.Error(fmt.Sprintf("Error processing InferRequest: %s", err.Error()))
		return &rayuserdefinedpb.CallResponse{}, err
	}

	return modelInferResponse, nil
}

func (r *ray) UpdateContainerizedModel(ctx context.Context, modelName string, userID string, imageName string, version string, hardware string, action Action, numOfGPU string) error {
	logger, _ := logx.GetZapLogger(ctx)

	var err error
	applicationMetadataValue := ""
	envVars := map[string]string{}

	if action != Sync {
		applicationMetadataValue, err = GetApplicationMetadataValue(modelName, version)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	if action == Deploy {
		envVars = r.setHardwareRunOptions(hardware, numOfGPU)
		if IsDummyModel(modelName) {
			envVars[EnvNumOfCPUs] = "0.001"
		}
		envVars[EnvNumOfMinReplicas] = "1"
		envVars[EnvNumOfMaxReplicas] = "10"
	}

	rayApplicationConfig := RayApplication{
		Name:        applicationMetadataValue,
		ImportPath:  "_model:entrypoint",
		RoutePrefix: "/" + applicationMetadataValue,
		RuntimeEnv: RuntimeEnv{
			ImageURI: fmt.Sprintf("%s:%v/%s/%s:%s", config.Config.Registry.Host, config.Config.Registry.Port, userID, imageName, version),
			EnvVars:  envVars,
		},
	}

	r.configChan <- ApplicationWithAction{
		RayApplication: rayApplicationConfig,
		Action:         action,
	}

	return <-r.doneChan
}

func (r *ray) setHardwareRunOptions(hardware string, numOfGPU string) map[string]string {
	logger, _ := logx.GetZapLogger(context.Background())

	envVars := map[string]string{}
	accelerator, ok := SupportedAcceleratorType[hardware]
	if !ok {
		logger.Warn("accelerator type(hardware) not supported, setting it as custom resource")
		return map[string]string{
			EnvTotalVRAM:         config.Config.Ray.Vram,
			EnvNumOfGPUs:         numOfGPU,
			EnvRayCustomResource: hardware,
		}
	}

	switch accelerator {
	case SupportedAcceleratorType["CPU"]:
		envVars[EnvNumOfCPUs] = "1"
	case SupportedAcceleratorType["GPU"]:
		envVars[EnvTotalVRAM] = config.Config.Ray.Vram
		envVars[EnvNumOfGPUs] = numOfGPU
	default:
		numOfGPUFloat, err := strconv.ParseFloat(numOfGPU, 64)
		if err != nil {
			numOfGPUFloat = 0
		}
		if numOfGPUFloat > 0 {
			envVars[EnvRayCustomResource] = hardware
			envVars[EnvNumOfGPUs] = numOfGPU
		} else {
			envVars[EnvRayCustomResource] = hardware
			envVars[EnvTotalVRAM] = strconv.Itoa(SupportedAcceleratorTypeMemory[hardware])
		}
	}

	return envVars
}

func (r *ray) sync() {
	for {
		applicationWithAction := <-r.configChan

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

		logger, _ := logx.GetZapLogger(ctx)

		if applicationWithAction.Action == UpScale {
			// this is a pseudo trigger request to invoke model upscale
			// we do not care about the trigger result
			go func() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("http://%s:%d%s", config.Config.Ray.Host, config.Config.Ray.Port.SERVE, applicationWithAction.RayApplication.RoutePrefix), http.NoBody)
				if err != nil {
					logger.Error(fmt.Sprintf("error while creating upscale request: %v", err))
					return
				}
				resp, err := r.httpClient.Do(req)
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

		newRayApplications := []RayApplication{}
		switch applicationWithAction.Action {
		case Deploy:
			for _, app := range modelDeploymentConfig.RayApplications {
				if app.Name != applicationWithAction.RayApplication.Name {
					newRayApplications = append(newRayApplications, app)
				}
			}
			modelDeploymentConfig.RayApplications = newRayApplications
			modelDeploymentConfig.RayApplications = append(modelDeploymentConfig.RayApplications, applicationWithAction.RayApplication)
		case Undeploy:
			for _, app := range modelDeploymentConfig.RayApplications {
				if app.Name != applicationWithAction.RayApplication.Name {
					newRayApplications = append(newRayApplications, app)
				}
			}
			modelDeploymentConfig.RayApplications = newRayApplications
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

		req, err := http.NewRequestWithContext(ctx, http.MethodPut, fmt.Sprintf("http://%s:%d/api/serve/applications/", config.Config.Ray.Host, config.Config.Ray.Port.DASHBOARD), bytes.NewBuffer(modelDeploymentConfigJSON))
		if err != nil {
			logger.Error(fmt.Sprintf("error while creating deployment request: %v", err))
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := r.httpClient.Do(req)
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

func (r *ray) Close() error {
	ctx := context.Background()

	logger, _ := logx.GetZapLogger(ctx)

	currentConfigFile, err := r.redisClient.Get(
		ctx,
		RayDeploymentKey,
	).Bytes()

	if err != nil {
		logger.Error(fmt.Sprintf("error while reading deployment config: %v", err))
		return err
	}
	if err := os.WriteFile(r.configFilePath, currentConfigFile, 0666); err != nil {
		logger.Error(fmt.Sprintf("error creating deployment config: %v", err))
		return err
	}

	if r.connection != nil {
		r.connection.Close()
	}
	close(r.configChan)
	close(r.doneChan)

	return nil
}
