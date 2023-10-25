package ray

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/logger"
	"github.com/instill-ai/model-backend/pkg/ray/rayserver"

	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
)

type InferInput interface{}

type TextToImageInput struct {
	Prompt   string
	Steps    int64
	CfgScale float32
	Seed     int64
	Samples  int64
}

type TextGenerationInput struct {
	Prompt        string
	OutputLen     int64
	BadWordsList  string
	StopWordsList string
	TopK          int64
	Seed          int64
}

type Ray interface {
	// grpc
	ModelReadyRequest(ctx context.Context, modelName string, modelInstance string) *rayserver.ModelReadyResponse
	ModelMetadataRequest(ctx context.Context, modelName string, modelInstance string) *rayserver.ModelMetadataResponse
	ModelInferRequest(ctx context.Context, task commonPB.Task, inferInput InferInput, modelName string, modelInstance string, modelMetadata *rayserver.ModelMetadataResponse) (*rayserver.ModelInferResponse, error)

	// standard
	IsRayServerReady(ctx context.Context) bool
	DeployModel(task commonPB.Task, modelPath string) error
	UndeployModel(modelPath string) error
	PostProcess(inferResponse *rayserver.ModelInferResponse, modelMetadata *rayserver.ModelMetadataResponse, task commonPB.Task) (interface{}, error)
	Close()
}

type ray struct {
	rayClient  rayserver.RayServiceClient
	connection *grpc.ClientConn
}

func NewRay() Ray {
	rayService := &ray{}
	rayService.Init()
	return rayService
}

func (r *ray) Init() {
	grpcUri := config.Config.RayServer.GrpcURI
	// Connect to gRPC server
	conn, err := grpc.Dial(grpcUri, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Couldn't connect to endpoint %s: %v", grpcUri, err)
	}

	// Create client from gRPC server connection
	r.connection = conn
	r.rayClient = rayserver.NewRayServiceClient(conn)
}

func (r *ray) IsRayServerReady(ctx context.Context) bool {
	return true
}

func (r *ray) ModelReadyRequest(ctx context.Context, modelName string, modelInstance string) *rayserver.ModelReadyResponse {
	logger, _ := logger.GetZapLogger(ctx)

	applicationMetadatValue, err := GetApplicationMetadaValue(modelName)
	if err != nil {
		logger.Error(err.Error())
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "application", applicationMetadatValue)

	// Create ready request for a given model
	modelReadyRequest := rayserver.ModelReadyRequest{
		Name:    modelName,
		Version: modelInstance,
	}

	// Submit modelReady request to server
	modelReadyResponse, err := r.rayClient.ModelReady(ctx, &modelReadyRequest)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			modelReadyResponse = &rayserver.ModelReadyResponse{
				Ready: false,
			}
		} else {
			logger.Error(err.Error())
		}
	}
	return modelReadyResponse
}

func (r *ray) ModelMetadataRequest(ctx context.Context, modelName string, modelInstance string) *rayserver.ModelMetadataResponse {

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

func (r *ray) ModelInferRequest(ctx context.Context, task commonPB.Task, inferInput InferInput, modelName string, modelInstance string, modelMetadata *rayserver.ModelMetadataResponse) (*rayserver.ModelInferResponse, error) {

	// Create request input tensors
	var inferInputs []*rayserver.InferTensor
	for i := 0; i < len(modelMetadata.Inputs); i++ {
		switch task {
		case commonPB.Task_TASK_TEXT_TO_IMAGE:
			inferInputs = append(inferInputs, &rayserver.InferTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{1},
			})
		case commonPB.Task_TASK_TEXT_GENERATION:
			inferInputs = append(inferInputs, &rayserver.InferTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{1, 1},
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
	var inferOutputs []*rayserver.ModelInferRequest_InferRequestedOutputTensor
	for i := 0; i < len(modelMetadata.Outputs); i++ {
		switch task {
		case commonPB.Task_TASK_CLASSIFICATION:
			inferOutputs = append(inferOutputs, &rayserver.ModelInferRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
			})
		case commonPB.Task_TASK_DETECTION:
			inferOutputs = append(inferOutputs, &rayserver.ModelInferRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
			})
		default:
			inferOutputs = append(inferOutputs, &rayserver.ModelInferRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
			})
		}
	}

	// Create inference request for specific model/version
	modelInferRequest := rayserver.ModelInferRequest{
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
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, SerializeBytesTensor([][]byte{[]byte(textToImageInput.Prompt)}))
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, SerializeBytesTensor([][]byte{[]byte("NONE")}))
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, samples)
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, SerializeBytesTensor([][]byte{[]byte("DPMSolverMultistepScheduler")})) // Fixed value.
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, steps)
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, guidanceScale)
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, seed)
	case commonPB.Task_TASK_TEXT_GENERATION:
		textGenerationInput := inferInput.(*TextGenerationInput)
		outputLen := make([]byte, 4)
		binary.LittleEndian.PutUint32(outputLen, uint32(textGenerationInput.OutputLen))
		topK := make([]byte, 4)
		binary.LittleEndian.PutUint32(topK, uint32(textGenerationInput.TopK))
		seed := make([]byte, 8)
		binary.LittleEndian.PutUint64(seed, uint64(textGenerationInput.Seed))
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, SerializeBytesTensor([][]byte{[]byte(textGenerationInput.Prompt)}))
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, outputLen)
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, SerializeBytesTensor([][]byte{[]byte(textGenerationInput.BadWordsList)}))
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, SerializeBytesTensor([][]byte{[]byte(textGenerationInput.StopWordsList)}))
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, topK)
		modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, seed)
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
	modelInferResponse, err := r.rayClient.ModelInfer(ctx, &modelInferRequest)
	if err != nil {
		log.Printf("Error processing InferRequest: %v", err)
		return &rayserver.ModelInferResponse{}, err
	}

	return modelInferResponse, nil
}

func (r *ray) PostProcess(inferResponse *rayserver.ModelInferResponse, modelMetadata *rayserver.ModelMetadataResponse, task commonPB.Task) (interface{}, error) {
	var (
		outputs interface{}
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

	case commonPB.Task_TASK_TEXT_TO_IMAGE:
		outputs, err = postProcessTextToImage(inferResponse, modelMetadata.Outputs[0].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process text to image output: %w", err)
		}

	case commonPB.Task_TASK_TEXT_GENERATION:
		outputs, err = postProcessTextGeneration(inferResponse, modelMetadata.Outputs[0].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process text to image output: %w", err)
		}

	default:
		outputs, err = postProcessUnspecifiedTask(inferResponse, modelMetadata.Outputs)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process unspecified output: %w", err)
		}
	}

	return outputs, nil
}

func (r *ray) DeployModel(task commonPB.Task, modelPath string) error {
	cmd := exec.Command("python", "ray_server.py",
		"--func", "deploy",
		"--task", task.String(),
		"--model", filepath.Join(config.Config.RayServer.ModelStore, modelPath),
	)
	cmd.Dir = FolderPath

	// var outb, errb bytes.Buffer
	// cmd.Stdout = &outb
	// cmd.Stderr = &errb
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	// fmt.Println("out:", outb.String(), "errb:", errb.String(), "err", err)

	return err
}

func (r *ray) UndeployModel(modelPath string) error {
	cmd := exec.Command("python", "ray_server.py",
		"--func", "undeploy",
		"--model", filepath.Join(config.Config.RayServer.ModelStore, modelPath),
	)
	cmd.Dir = FolderPath

	// var outb, errb bytes.Buffer
	// cmd.Stdout = &outb
	// cmd.Stderr = &errb
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	// fmt.Println("out:", outb.String(), "errb:", errb.String(), "err", err)

	return err
}

func (r *ray) Close() {
	if r.connection != nil {
		r.connection.Close()
	}
}
