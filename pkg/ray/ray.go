package ray

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/ray/rayserver"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

type Ray interface {
	// grpc
	ModelReady(ctx context.Context, modelName string, modelInstance string) (*modelPB.Model_State, error)
	ModelMetadataRequest(ctx context.Context, modelName string, modelInstance string) *rayserver.ModelMetadataResponse
	ModelInferRequest(ctx context.Context, task commonPB.Task, inferInput InferInput, modelName string, modelInstance string, modelMetadata *rayserver.ModelMetadataResponse) (*rayserver.RayServiceCallResponse, error)

	// standard
	IsRayServerReady(ctx context.Context) bool
	DeployModel(modelPath string) error
	UndeployModel(modelPath string) error
	Init()
	Close()
}

type ray struct {
	rayClient      rayserver.RayServiceClient
	rayServeClient rayserver.RayServeAPIServiceClient
	rayHTTPClient  *http.Client
	connection     *grpc.ClientConn
}

func NewRay() Ray {
	rayService := &ray{}
	rayService.Init()
	return rayService
}

func (r *ray) Init() {
	grpcURI := config.Config.RayServer.GrpcURI
	// Connect to gRPC server
	conn, err := grpc.Dial(
		grpcURI,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.Config.Server.MaxDataSize*constant.MB),
			grpc.MaxCallSendMsgSize(config.Config.Server.MaxDataSize*constant.MB),
		),
	)

	if err != nil {
		log.Fatalf("Couldn't connect to endpoint %s: %v", grpcURI, err)
	}

	// Create client from gRPC server connection
	r.connection = conn
	r.rayClient = rayserver.NewRayServiceClient(conn)
	r.rayServeClient = rayserver.NewRayServeAPIServiceClient(conn)
	r.rayHTTPClient = &http.Client{Timeout: time.Second * 5}
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

func (r *ray) ModelReady(ctx context.Context, modelName string, modelInstance string) (*modelPB.Model_State, error) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.Replace(fmt.Sprintf("http://%s/api/serve/applications/", config.Config.RayServer.GrpcURI), "9000", "8265", 1), http.NoBody)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	var applicationStatus rayserver.GetApplicationStatus
	err = json.NewDecoder(resp.Body).Decode(&applicationStatus)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	deployment, ok := applicationStatus.Applications[applicationMetadatValue]
	if !ok {
		return modelPB.Model_STATE_OFFLINE.Enum(), nil
	}

	switch deployment.Status {
	case "RUNNING":
		return modelPB.Model_STATE_ONLINE.Enum(), nil
	case "DEPLOY_FAILED":
	case "UNHEALTHY":
		return modelPB.Model_STATE_ERROR.Enum(), nil
	case "DEPLOYING":
	case "DELETING":
		return modelPB.Model_STATE_UNSPECIFIED.Enum(), nil
	case "NOT_STARTED":
		return modelPB.Model_STATE_OFFLINE.Enum(), nil
	}

	return modelPB.Model_STATE_ERROR.Enum(), nil
}

func (r *ray) ModelMetadataRequest(ctx context.Context, modelName string, modelInstance string) *rayserver.ModelMetadataResponse {
	logger, _ := custom_logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName)
	if err != nil {
		logger.Error(err.Error())
		return nil
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "application", applicationMetadatValue)

	// Create status request for a given model
	modelMetadataRequest := rayserver.ModelMetadataRequest{
		Name:    modelName,
		Version: modelInstance,
	}
	// Submit modelMetadata request to server
	modelMetadataResponse, err := r.rayClient.ModelMetadata(ctx, &modelMetadataRequest)
	if err != nil {
		log.Printf("Couldn't get server model metadata: %v", err)
	}
	return modelMetadataResponse
}

func (r *ray) ModelInferRequest(ctx context.Context, task commonPB.Task, inferInput InferInput, modelName string, modelInstance string, modelMetadata *rayserver.ModelMetadataResponse) (*rayserver.RayServiceCallResponse, error) {
	logger, _ := custom_logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "application", applicationMetadatValue)

	// Create request input tensors
	var inferInputs []*rayserver.InferTensor
	for i := 0; i < len(modelMetadata.Inputs); i++ {
		switch task {
		case commonPB.Task_TASK_IMAGE_TO_IMAGE,
			commonPB.Task_TASK_TEXT_TO_IMAGE:
			inferInputs = append(inferInputs, &rayserver.InferTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{1},
			})
		case commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING,
			commonPB.Task_TASK_TEXT_GENERATION_CHAT,
			commonPB.Task_TASK_TEXT_GENERATION:
			var inputShape []int64
			inputShape = []int64{1}

			inferInputs = append(inferInputs, &rayserver.InferTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    inputShape,
			})
		case commonPB.Task_TASK_CLASSIFICATION,
			commonPB.Task_TASK_DETECTION,
			commonPB.Task_TASK_KEYPOINT,
			commonPB.Task_TASK_OCR,
			commonPB.Task_TASK_INSTANCE_SEGMENTATION,
			commonPB.Task_TASK_SEMANTIC_SEGMENTATION:
			batchSize := int64(len(inferInput.([][]byte)))
			inferInputs = append(inferInputs, &rayserver.InferTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{batchSize, 1},
			})
		default:
			batchSize := int64(len(inferInput.([][]byte)))
			inferInputs = append(inferInputs, &rayserver.InferTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{batchSize, 1},
			})
		}
	}

	// Create request input output tensors
	var inferOutputs []*rayserver.RayServiceCallRequest_InferRequestedOutputTensor
	for i := 0; i < len(modelMetadata.Outputs); i++ {
		switch task {
		case commonPB.Task_TASK_CLASSIFICATION:
			inferOutputs = append(inferOutputs, &rayserver.RayServiceCallRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
			})
		case commonPB.Task_TASK_DETECTION:
			inferOutputs = append(inferOutputs, &rayserver.RayServiceCallRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
			})
		default:
			inferOutputs = append(inferOutputs, &rayserver.RayServiceCallRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
			})
		}
	}

	// Create inference request for specific model/version
	modelInferRequest := rayserver.RayServiceCallRequest{
		ModelName:    modelName,
		ModelVersion: modelInstance,
		Inputs:       inferInputs,
		Outputs:      inferOutputs,
	}

	switch task {
	case commonPB.Task_TASK_TEXT_TO_IMAGE:
		textToImageInput := inferInput.(*TextToImageInput)
		samples := make([]byte, 4)
		binary.LittleEndian.PutUint32(samples, uint32(textToImageInput.Samples))
		steps := make([]byte, 4)
		binary.LittleEndian.PutUint32(steps, uint32(textToImageInput.Steps))
		guidanceScale := make([]byte, 4)
		binary.LittleEndian.PutUint32(guidanceScale, math.Float32bits(textToImageInput.CfgScale)) // Fixed value.
		seed := make([]byte, 8)
		binary.LittleEndian.PutUint64(seed, uint64(textToImageInput.Seed))
		modelInferRequest.RawInputContents = append(
			modelInferRequest.RawInputContents,
			SerializeBytesTensor([][]byte{[]byte(textToImageInput.Prompt)}),
			SerializeBytesTensor([][]byte{[]byte("NONE")}),
			SerializeBytesTensor([][]byte{[]byte(textToImageInput.PromptImage)}),
			samples,
			SerializeBytesTensor([][]byte{[]byte("DPMSolverMultistepScheduler")}), // Fixed value
			steps,
			guidanceScale,
			seed,
			SerializeBytesTensor([][]byte{[]byte(textToImageInput.ExtraParams)}),
		)
	case commonPB.Task_TASK_IMAGE_TO_IMAGE:
		imageToImageInput := inferInput.(*ImageToImageInput)
		samples := make([]byte, 4)
		binary.LittleEndian.PutUint32(samples, uint32(imageToImageInput.Samples))
		steps := make([]byte, 4)
		binary.LittleEndian.PutUint32(steps, uint32(imageToImageInput.Steps))
		guidanceScale := make([]byte, 4)
		binary.LittleEndian.PutUint32(guidanceScale, math.Float32bits(imageToImageInput.CfgScale)) // Fixed value.
		seed := make([]byte, 8)
		binary.LittleEndian.PutUint64(seed, uint64(imageToImageInput.Seed))
		modelInferRequest.RawInputContents = append(
			modelInferRequest.RawInputContents,
			SerializeBytesTensor([][]byte{[]byte(imageToImageInput.Prompt)}),
			SerializeBytesTensor([][]byte{[]byte("NONE")}),
			SerializeBytesTensor([][]byte{[]byte(imageToImageInput.PromptImage)}),
			samples,
			SerializeBytesTensor([][]byte{[]byte("DPMSolverMultistepScheduler")}), // Fixed value,
			steps,
			guidanceScale,
			seed,
			SerializeBytesTensor([][]byte{[]byte(imageToImageInput.ExtraParams)}),
		)
	case commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING:
		visualQUestionAnsweringInput := inferInput.(*VisualQuestionAnsweringInput)
		maxNewToken := make([]byte, 4)
		binary.LittleEndian.PutUint32(maxNewToken, uint32(visualQUestionAnsweringInput.MaxNewTokens))
		temperature := make([]byte, 4)
		binary.LittleEndian.PutUint32(temperature, math.Float32bits(visualQUestionAnsweringInput.Temperature))
		topK := make([]byte, 4)
		binary.LittleEndian.PutUint32(topK, uint32(visualQUestionAnsweringInput.TopK))
		seed := make([]byte, 8)
		binary.LittleEndian.PutUint64(seed, uint64(visualQUestionAnsweringInput.Seed))
		modelInferRequest.RawInputContents = append(
			modelInferRequest.RawInputContents,
			SerializeBytesTensor([][]byte{[]byte(visualQUestionAnsweringInput.Prompt)}),
			SerializeBytesTensor([][]byte{[]byte(visualQUestionAnsweringInput.PromptImages)}),
			SerializeBytesTensor([][]byte{[]byte(visualQUestionAnsweringInput.ChatHistory)}),
			SerializeBytesTensor([][]byte{[]byte(visualQUestionAnsweringInput.SystemMessage)}),
			maxNewToken,
			temperature,
			topK,
			seed,
			SerializeBytesTensor([][]byte{[]byte(visualQUestionAnsweringInput.ExtraParams)}),
		)
	case commonPB.Task_TASK_TEXT_GENERATION_CHAT:
		textGenerationChatInput := inferInput.(*TextGenerationChatInput)
		maxNewToken := make([]byte, 4)
		binary.LittleEndian.PutUint32(maxNewToken, uint32(textGenerationChatInput.MaxNewTokens))
		temperature := make([]byte, 4)
		binary.LittleEndian.PutUint32(temperature, math.Float32bits(textGenerationChatInput.Temperature))
		topK := make([]byte, 4)
		binary.LittleEndian.PutUint32(topK, uint32(textGenerationChatInput.TopK))
		seed := make([]byte, 8)
		binary.LittleEndian.PutUint64(seed, uint64(textGenerationChatInput.Seed))
		modelInferRequest.RawInputContents = append(
			modelInferRequest.RawInputContents,
			SerializeBytesTensor([][]byte{[]byte(textGenerationChatInput.Prompt)}),
			SerializeBytesTensor([][]byte{[]byte(textGenerationChatInput.PromptImages)}),
			SerializeBytesTensor([][]byte{[]byte(textGenerationChatInput.ChatHistory)}),
			SerializeBytesTensor([][]byte{[]byte(textGenerationChatInput.SystemMessage)}),
			maxNewToken,
			temperature,
			topK,
			seed,
			SerializeBytesTensor([][]byte{[]byte(textGenerationChatInput.ExtraParams)}),
		)
	case commonPB.Task_TASK_TEXT_GENERATION:
		textGenerationInput := inferInput.(*TextGenerationInput)
		maxNewToken := make([]byte, 4)
		binary.LittleEndian.PutUint32(maxNewToken, uint32(textGenerationInput.MaxNewTokens))
		temperature := make([]byte, 4)
		binary.LittleEndian.PutUint32(temperature, math.Float32bits(textGenerationInput.Temperature))
		topK := make([]byte, 4)
		binary.LittleEndian.PutUint32(topK, uint32(textGenerationInput.TopK))
		seed := make([]byte, 8)
		binary.LittleEndian.PutUint64(seed, uint64(textGenerationInput.Seed))
		modelInferRequest.RawInputContents = append(
			modelInferRequest.RawInputContents,
			SerializeBytesTensor([][]byte{[]byte(textGenerationInput.Prompt)}),
			SerializeBytesTensor([][]byte{[]byte(textGenerationInput.PromptImages)}),
			SerializeBytesTensor([][]byte{[]byte(textGenerationInput.ChatHistory)}),
			SerializeBytesTensor([][]byte{[]byte(textGenerationInput.SystemMessage)}),
			maxNewToken,
			temperature,
			topK,
			seed,
			SerializeBytesTensor([][]byte{[]byte(textGenerationInput.ExtraParams)}),
		)
	case commonPB.Task_TASK_CLASSIFICATION,
		commonPB.Task_TASK_DETECTION,
		commonPB.Task_TASK_KEYPOINT,
		commonPB.Task_TASK_OCR,
		commonPB.Task_TASK_INSTANCE_SEGMENTATION,
		commonPB.Task_TASK_SEMANTIC_SEGMENTATION:
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, SerializeBytesTensor(inferInput.([][]byte)))
	default:
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, SerializeBytesTensor(inferInput.([][]byte)))
	}

	// Submit inference request to server
	modelInferResponse, err := r.rayClient.XCall__(ctx, &modelInferRequest)
	if err != nil {
		logger.Error(fmt.Sprintf("Error processing InferRequest: %s", err.Error()))
		return &rayserver.RayServiceCallResponse{}, err
	}

	return modelInferResponse, nil
}

func PostProcess(inferResponse *rayserver.RayServiceCallResponse, modelMetadata *rayserver.ModelMetadataResponse, task commonPB.Task) (any, error) {
	var (
		outputs any
		err     error
	)

	switch task {
	case commonPB.Task_TASK_CLASSIFICATION:
		outputs, err = postProcessClassification(inferResponse, modelMetadata.Outputs[0].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process classification output: %w", err)
		}
	case commonPB.Task_TASK_DETECTION:
		if len(modelMetadata.Outputs) < 2 {
			return nil, fmt.Errorf("wrong output format of detection task")
		}
		outputs, err = postProcessDetection(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process detection output: %w", err)
		}
	case commonPB.Task_TASK_KEYPOINT:
		if len(modelMetadata.Outputs) < 3 {
			return nil, fmt.Errorf("wrong output format of keypoint detection task")
		}
		outputs, err = postProcessKeypoint(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name, modelMetadata.Outputs[2].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process keypoint output: %w", err)
		}
	case commonPB.Task_TASK_OCR:
		if len(modelMetadata.Outputs) < 2 {
			return nil, fmt.Errorf("wrong output format of OCR task")
		}
		switch len(modelMetadata.Outputs) {
		case 2:
			outputs, err = postProcessOcrWithoutScore(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name)
			if err != nil {
				return nil, fmt.Errorf("unable to post-process detection output: %w", err)
			}
		case 3:
			outputs, err = postProcessOcrWithScore(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name, modelMetadata.Outputs[2].Name)
			if err != nil {
				return nil, fmt.Errorf("unable to post-process detection output: %w", err)
			}
		}

	case commonPB.Task_TASK_INSTANCE_SEGMENTATION:
		if len(modelMetadata.Outputs) < 4 {
			return nil, fmt.Errorf("wrong output format of instance segmentation task")
		}
		outputs, err = postProcessInstanceSegmentation(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name, modelMetadata.Outputs[2].Name, modelMetadata.Outputs[3].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process instance segmentation output: %w", err)
		}

	case commonPB.Task_TASK_SEMANTIC_SEGMENTATION:
		if len(modelMetadata.Outputs) < 2 {
			return nil, fmt.Errorf("wrong output format of semantic segmentation task")
		}
		outputs, err = postProcessSemanticSegmentation(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process semantic segmentation output: %w", err)
		}

	case commonPB.Task_TASK_IMAGE_TO_IMAGE,
		commonPB.Task_TASK_TEXT_TO_IMAGE:
		outputs, err = postProcessTextToImage(inferResponse, modelMetadata.Outputs[0].Name, task)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process text to image output: %w", err)
		}

	case commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING,
		commonPB.Task_TASK_TEXT_GENERATION_CHAT,
		commonPB.Task_TASK_TEXT_GENERATION:
		outputs, err = postProcessTextGeneration(inferResponse, modelMetadata.Outputs[0].Name, task)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process text to text output: %w", err)
		}

	default:
		outputs, err = postProcessUnspecifiedTask(inferResponse, modelMetadata.Outputs)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process unspecified output: %w", err)
		}
	}

	return outputs, nil
}

func (r *ray) DeployModel(modelPath string) error {
	modelPath = filepath.Join(config.Config.RayServer.ModelStore, modelPath)
	cmd := exec.Command("/ray-conda/bin/python", "-c", fmt.Sprintf("from model import deployable; deployable.deploy(%q, %q, %q)", modelPath, config.Config.RayServer.GrpcURI, config.Config.RayServer.Vram))
	cmd.Dir = modelPath

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	return err
}

func (r *ray) UndeployModel(modelPath string) error {
	modelPath = filepath.Join(config.Config.RayServer.ModelStore, modelPath)
	cmd := exec.Command("/ray-conda/bin/python", "-c", fmt.Sprintf("from model import deployable; deployable.undeploy(%q, %q)", modelPath, config.Config.RayServer.GrpcURI))
	cmd.Dir = modelPath

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	return err
}

func (r *ray) Close() {
	if r.connection != nil {
		r.connection.Close()
	}
}
