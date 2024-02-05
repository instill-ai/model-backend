// Inspired from https://github.com/triton-inference-server/server/blob/v2.5.0/src/clients/go/grpc_simple_client.go

package triton

import (
	"context"
	"encoding/binary"
	"log"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/instill-ai/model-backend/config"
	"github.com/instill-ai/model-backend/pkg/constant"
	"github.com/instill-ai/model-backend/pkg/triton/inferenceserver"

	custom_logger "github.com/instill-ai/model-backend/pkg/logger"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
)

type InferInput any

type TextToImageInput struct {
	Prompt      string
	PromptImage string
	Steps       int32
	CfgScale    float32
	Seed        int32
	Samples     int32
	ExtraParams string
}

type ImageToImageInput struct {
	Prompt      string
	PromptImage string
	Steps       int32
	CfgScale    float32
	Seed        int32
	Samples     int32
	ExtraParams string
}

type TextGenerationInput struct {
	Prompt        string
	PromptImages  string
	ChatHistory   string
	SystemMessage string
	MaxNewTokens  int32
	Temperature   float32
	TopK          int32
	Seed          int32
	ExtraParams   string
}

type TextGenerationChatInput struct {
	Prompt        string
	PromptImages  string
	ChatHistory   string
	SystemMessage string
	MaxNewTokens  int32
	Temperature   float32
	TopK          int32
	Seed          int32
	ExtraParams   string
}

type VisualQuestionAnsweringInput struct {
	Prompt        string
	PromptImages  string
	ChatHistory   string
	SystemMessage string
	MaxNewTokens  int32
	Temperature   float32
	TopK          int32
	Seed          int32
	ExtraParams   string
}

type ImageInput struct {
	ImgURL    string
	ImgBase64 string
}

type Triton interface {
	ServerLiveRequest(ctx context.Context) *inferenceserver.ServerLiveResponse
	ServerReadyRequest(ctx context.Context) *inferenceserver.ServerReadyResponse
	ModelReadyRequest(ctx context.Context, modelName string, modelInstance string) *inferenceserver.ModelReadyResponse
	ModelMetadataRequest(ctx context.Context, modelName string, modelInstance string) *inferenceserver.ModelMetadataResponse
	ModelConfigRequest(ctx context.Context, modelName string, modelInstance string) *inferenceserver.ModelConfigResponse
	ModelInferRequest(ctx context.Context, task commonPB.Task, inferInput InferInput, modelName string, modelInstance string, modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse) (*inferenceserver.ModelInferResponse, error)
	LoadModelRequest(ctx context.Context, modelName string) (*inferenceserver.RepositoryModelLoadResponse, error)
	UnloadModelRequest(ctx context.Context, modelName string) (*inferenceserver.RepositoryModelUnloadResponse, error)
	ListModelsRequest(ctx context.Context) *inferenceserver.RepositoryIndexResponse
	IsTritonServerReady(ctx context.Context) bool
	Init()
	Close()
}

type triton struct {
	tritonClient inferenceserver.GRPCInferenceServiceClient
	connection   *grpc.ClientConn
}

func NewTriton() Triton {
	tritonService := &triton{}
	tritonService.Init()
	return tritonService
}

func (ts *triton) Init() {
	grpcURI := config.Config.TritonServer.GrpcURI

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
	ts.connection = conn
	ts.tritonClient = inferenceserver.NewGRPCInferenceServiceClient(conn)
}

func (ts *triton) Close() {
	if ts.connection != nil {
		ts.connection.Close()
	}
}

func (ts *triton) ServerLiveRequest(ctx context.Context) *inferenceserver.ServerLiveResponse {

	serverLiveRequest := inferenceserver.ServerLiveRequest{}
	// Submit ServerLive request to server
	serverLiveResponse, err := ts.tritonClient.ServerLive(ctx, &serverLiveRequest)
	if err != nil {
		log.Printf("Couldn't get server live: %v", err)
	}
	return serverLiveResponse
}

func (ts *triton) ServerReadyRequest(ctx context.Context) *inferenceserver.ServerReadyResponse {

	serverReadyRequest := inferenceserver.ServerReadyRequest{}
	// Submit ServerReady request to server
	serverReadyResponse, err := ts.tritonClient.ServerReady(ctx, &serverReadyRequest)
	if err != nil {
		log.Printf("Couldn't get server ready: %v", err)
	}
	return serverReadyResponse
}

func (ts *triton) ModelReadyRequest(ctx context.Context, modelName string, modelInstance string) *inferenceserver.ModelReadyResponse {
	logger, _ := custom_logger.GetZapLogger(ctx)

	// Create ready request for a given model
	modelReadyRequest := inferenceserver.ModelReadyRequest{
		Name:    modelName,
		Version: modelInstance,
	}

	// Submit modelReady request to server
	modelReadyResponse, err := ts.tritonClient.ModelReady(ctx, &modelReadyRequest)

	if err != nil {
		logger.Error(err.Error())
	}
	return modelReadyResponse
}

func (ts *triton) ModelMetadataRequest(ctx context.Context, modelName string, modelInstance string) *inferenceserver.ModelMetadataResponse {

	// Create status request for a given model
	modelMetadataRequest := inferenceserver.ModelMetadataRequest{
		Name:    modelName,
		Version: modelInstance,
	}
	// Submit modelMetadata request to server
	modelMetadataResponse, err := ts.tritonClient.ModelMetadata(ctx, &modelMetadataRequest)
	if err != nil {
		log.Printf("Couldn't get server model metadata: %v", err)
	}
	return modelMetadataResponse
}

func (ts *triton) ModelConfigRequest(ctx context.Context, modelName string, modelInstance string) *inferenceserver.ModelConfigResponse {

	// Create status request for a given model
	modelConfigRequest := inferenceserver.ModelConfigRequest{
		Name:    modelName,
		Version: modelInstance,
	}
	// Submit modelMetadata request to server
	modelConfigResponse, err := ts.tritonClient.ModelConfig(ctx, &modelConfigRequest)
	if err != nil {
		log.Printf("Couldn't get server model config: %v", err)
	}
	return modelConfigResponse
}

func (ts *triton) ModelInferRequest(ctx context.Context, task commonPB.Task, inferInput InferInput, modelName string, modelInstance string, modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse) (*inferenceserver.ModelInferResponse, error) {
	// Create request input tensors
	var inferInputs []*inferenceserver.ModelInferRequest_InferInputTensor
	for i := 0; i < len(modelMetadata.Inputs); i++ {
		switch task {
		case commonPB.Task_TASK_IMAGE_TO_IMAGE,
			commonPB.Task_TASK_TEXT_TO_IMAGE:
			inferInputs = append(inferInputs, &inferenceserver.ModelInferRequest_InferInputTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{1},
			})
		case commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING,
			commonPB.Task_TASK_TEXT_GENERATION_CHAT,
			commonPB.Task_TASK_TEXT_GENERATION:
			var inputShape []int64
			if modelConfig.Config.MaxBatchSize > 0 {
				inputShape = []int64{1, 1}
			} else {
				inputShape = []int64{1}
			}

			inferInputs = append(inferInputs, &inferenceserver.ModelInferRequest_InferInputTensor{
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
			if modelConfig.Config.Platform == "ensemble" {
				inferInputs = append(inferInputs, &inferenceserver.ModelInferRequest_InferInputTensor{
					Name:     modelMetadata.Inputs[i].Name,
					Datatype: modelMetadata.Inputs[i].Datatype,
					Shape:    []int64{batchSize, 1},
				})
			} else {
				c, h, w := ParseModel(modelMetadata, modelConfig)
				var shape []int64
				if modelConfig.Config.Input[0].Format == 1 { //Format::FORMAT_NHWC = 1
					shape = []int64{1, h, w, c}
				} else {
					shape = []int64{1, c, h, w}
				}
				inferInputs = append(inferInputs, &inferenceserver.ModelInferRequest_InferInputTensor{
					Name:     modelMetadata.Inputs[i].Name,
					Datatype: modelMetadata.Inputs[i].Datatype,
					Shape:    shape,
				})
			}
		default:
			batchSize := int64(len(inferInput.([][]byte)))
			inferInputs = append(inferInputs, &inferenceserver.ModelInferRequest_InferInputTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{batchSize, 1},
			})
		}
	}

	// Create request input output tensors
	var inferOutputs []*inferenceserver.ModelInferRequest_InferRequestedOutputTensor
	for i := 0; i < len(modelMetadata.Outputs); i++ {
		switch task {
		case commonPB.Task_TASK_CLASSIFICATION:
			inferOutputs = append(inferOutputs, &inferenceserver.ModelInferRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
				Parameters: map[string]*inferenceserver.InferParameter{
					"classification": {
						ParameterChoice: &inferenceserver.InferParameter_Int64Param{
							Int64Param: 1,
						},
					},
				},
			})
		case commonPB.Task_TASK_DETECTION:
			inferOutputs = append(inferOutputs, &inferenceserver.ModelInferRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
			})
		default:
			inferOutputs = append(inferOutputs, &inferenceserver.ModelInferRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
			})
		}
	}

	// Create inference request for specific model/version
	modelInferRequest := inferenceserver.ModelInferRequest{
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
		if textToImageInput.CfgScale > math.MaxFloat32 {
			textToImageInput.CfgScale = math.MaxFloat32
		}
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
		if imageToImageInput.CfgScale > math.MaxFloat32 {
			imageToImageInput.CfgScale = math.MaxFloat32
		}
		binary.LittleEndian.PutUint32(guidanceScale, math.Float32bits(imageToImageInput.CfgScale)) // Fixed value.
		seed := make([]byte, 8)
		binary.LittleEndian.PutUint64(seed, uint64(imageToImageInput.Seed))
		modelInferRequest.RawInputContents = append(
			modelInferRequest.RawInputContents,
			SerializeBytesTensor([][]byte{[]byte(imageToImageInput.Prompt)}),
			SerializeBytesTensor([][]byte{[]byte("NONE")}),
			SerializeBytesTensor([][]byte{[]byte(imageToImageInput.PromptImage)}),
			samples,
			SerializeBytesTensor([][]byte{[]byte("DPMSolverMultistepScheduler")}), // Fixed value
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
	modelInferResponse, err := ts.tritonClient.ModelInfer(ctx, &modelInferRequest)
	if err != nil {
		log.Printf("Error processing InferRequest: %v", err)
		return &inferenceserver.ModelInferResponse{}, err
	}
	return modelInferResponse, nil
}

func (ts *triton) LoadModelRequest(ctx context.Context, modelName string) (*inferenceserver.RepositoryModelLoadResponse, error) {

	// Create status request for a given model
	loadModelRequest := inferenceserver.RepositoryModelLoadRequest{
		RepositoryName: "",
		ModelName:      modelName,
	}
	// Submit loadModelRequest request to server
	return ts.tritonClient.RepositoryModelLoad(ctx, &loadModelRequest)
}

func (ts *triton) UnloadModelRequest(ctx context.Context, modelName string) (*inferenceserver.RepositoryModelUnloadResponse, error) {

	// Create status request for a given model
	unloadModelRequest := inferenceserver.RepositoryModelUnloadRequest{
		RepositoryName: "",
		ModelName:      modelName,
	}
	// Submit loadModelRequest request to server
	return ts.tritonClient.RepositoryModelUnload(ctx, &unloadModelRequest)
}

func (ts *triton) ListModelsRequest(ctx context.Context) *inferenceserver.RepositoryIndexResponse {

	// Create status request for a given model
	listModelsRequest := inferenceserver.RepositoryIndexRequest{
		RepositoryName: "",
	}
	// Submit loadModelRequest request to server
	listModelsResponse, err := ts.tritonClient.RepositoryIndex(ctx, &listModelsRequest)
	if err != nil {
		log.Printf("Couldn't list models: %v", err)
	}
	return listModelsResponse
}

func (ts *triton) IsTritonServerReady(ctx context.Context) bool {
	serverLiveResponse := ts.ServerLiveRequest(ctx)
	if serverLiveResponse == nil {
		return false
	}
	// fmt.Printf("Triton Health - Live: %v\n", serverLiveResponse.Live)
	return serverLiveResponse.Live
}
