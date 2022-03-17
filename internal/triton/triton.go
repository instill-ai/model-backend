// Inspired from https://github.com/triton-inference-server/server/blob/v2.5.0/src/clients/go/grpc_simple_client.go

package triton

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/instill-ai/model-backend/configs"
	"github.com/instill-ai/model-backend/internal/inferenceserver"
	model "github.com/instill-ai/protogen-go/model/v1alpha"
	"google.golang.org/grpc"
)

type TritonService interface {
	ServerLiveRequest() *inferenceserver.ServerLiveResponse
	ServerReadyRequest() *inferenceserver.ServerReadyResponse
	ModelMetadataRequest(modelName string, modelVersion string) *inferenceserver.ModelMetadataResponse
	ModelConfigRequest(modelName string, modelVersion string) *inferenceserver.ModelConfigResponse
	ModelInferRequest(cvTask model.Model_Task, rawInput [][]byte, modelName string, modelVersion string, modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse) (*inferenceserver.ModelInferResponse, error)
	PostProcess(inferResponse *inferenceserver.ModelInferResponse, modelMetadata *inferenceserver.ModelMetadataResponse, task model.Model_Task) (interface{}, error)
	LoadModelRequest(modelName string) (*inferenceserver.RepositoryModelLoadResponse, error)
	UnloadModelRequest(modelName string) (*inferenceserver.RepositoryModelUnloadResponse, error)
	ListModelsRequest() *inferenceserver.RepositoryIndexResponse
	IsTritonServerReady() bool
	Init()
	Close()
}

type tritonService struct {
	tritonClient inferenceserver.GRPCInferenceServiceClient
	connection   *grpc.ClientConn
}

func NewTritonService() TritonService {
	tritonService := &tritonService{}
	tritonService.Init()
	return tritonService
}

func (ts *tritonService) Init() {
	grpcUri := configs.Config.TritonServer.GrpcUri
	// Connect to gRPC server
	conn, err := grpc.Dial(grpcUri, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Couldn't connect to endpoint %s: %v", grpcUri, err)
	}

	// Create client from gRPC server connection
	ts.connection = conn
	ts.tritonClient = inferenceserver.NewGRPCInferenceServiceClient(conn)
}

func (ts *tritonService) Close() {
	if ts.connection != nil {
		ts.connection.Close()
	}
}

func (ts *tritonService) ServerLiveRequest() *inferenceserver.ServerLiveResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	serverLiveRequest := inferenceserver.ServerLiveRequest{}
	// Submit ServerLive request to server
	serverLiveResponse, err := ts.tritonClient.ServerLive(ctx, &serverLiveRequest)
	if err != nil {
		log.Printf("Couldn't get server live: %v", err)
	}
	return serverLiveResponse
}

func (ts *tritonService) ServerReadyRequest() *inferenceserver.ServerReadyResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	serverReadyRequest := inferenceserver.ServerReadyRequest{}
	// Submit ServerReady request to server
	serverReadyResponse, err := ts.tritonClient.ServerReady(ctx, &serverReadyRequest)
	if err != nil {
		log.Printf("Couldn't get server ready: %v", err)
	}
	return serverReadyResponse
}

func (ts *tritonService) ModelMetadataRequest(modelName string, modelVersion string) *inferenceserver.ModelMetadataResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create status request for a given model
	modelMetadataRequest := inferenceserver.ModelMetadataRequest{
		Name:    modelName,
		Version: modelVersion,
	}
	fmt.Println(">>>>>>>>chhcccc ", ts.tritonClient)
	// Submit modelMetadata request to server
	modelMetadataResponse, err := ts.tritonClient.ModelMetadata(ctx, &modelMetadataRequest)
	if err != nil {
		log.Printf("Couldn't get server model metadata: %v", err)
	}
	return modelMetadataResponse
}

func (ts *tritonService) ModelConfigRequest(modelName string, modelVersion string) *inferenceserver.ModelConfigResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create status request for a given model
	modelConfigRequest := inferenceserver.ModelConfigRequest{
		Name:    modelName,
		Version: modelVersion,
	}
	// Submit modelMetadata request to server
	modelConfigResponse, err := ts.tritonClient.ModelConfig(ctx, &modelConfigRequest)
	if err != nil {
		log.Printf("Couldn't get server model config: %v", err)
	}
	return modelConfigResponse
}

func (ts *tritonService) ModelInferRequest(cvTask model.Model_Task, rawInput [][]byte, modelName string, modelVersion string, modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse) (*inferenceserver.ModelInferResponse, error) {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create request input tensors
	batchSize := int64(len(rawInput))
	var inferInputs []*inferenceserver.ModelInferRequest_InferInputTensor
	for i := 0; i < len(modelMetadata.Inputs); i++ {
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
	}

	// Create request input output tensors
	var inferOutputs []*inferenceserver.ModelInferRequest_InferRequestedOutputTensor
	for i := 0; i < len(modelMetadata.Outputs); i++ {
		switch cvTask {
		case model.Model_TASK_CLASSIFICATION:
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
		case model.Model_TASK_DETECTION:
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
		ModelVersion: modelVersion,
		Inputs:       inferInputs,
		Outputs:      inferOutputs,
	}
	modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, SerializeBytesTensor(rawInput))

	// Submit inference request to server
	modelInferResponse, err := ts.tritonClient.ModelInfer(ctx, &modelInferRequest)
	if err != nil {
		log.Printf("Error processing InferRequest: %v", err)
		return &inferenceserver.ModelInferResponse{}, err
	}

	return modelInferResponse, nil
}

func postProcessDetection(modelInferResponse *inferenceserver.ModelInferResponse, outputNameBboxes string, outputNameLabels string) (interface{}, error) {
	outputTensorBboxes, rawOutputContentBboxes, err := GetOutputFromInferResponse(outputNameBboxes, modelInferResponse)

	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("Unable to find inference output for boxes")
	}
	if rawOutputContentBboxes == nil {
		return nil, fmt.Errorf("Unable to find output content for boxes")
	}
	outputTensorLabels, rawOutputContentLabels, err := GetOutputFromInferResponse(outputNameLabels, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("Unable to find inference output for labels")
	}
	if rawOutputContentLabels == nil {
		return nil, fmt.Errorf("Unable to find output content for labels")
	}

	outputDataBboxes := DeserializeFloat32Tensor(rawOutputContentBboxes)
	batchedOutputDataBboxes, err := Reshape1DArrayFloat32To3D(outputDataBboxes, outputTensorBboxes.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("Unable to reshape inference output for boxes")
	}

	outputDataLabels := DeserializeBytesTensor(rawOutputContentLabels, outputTensorBboxes.Shape[0]*outputTensorBboxes.Shape[1])
	batchedOutputDataLabels, err := Reshape1DArrayStringTo2D(outputDataLabels, outputTensorLabels.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("Unable to reshape inference output for labels")
	}

	if len(batchedOutputDataBboxes) != len(batchedOutputDataLabels) {
		log.Printf("Bboxes output has length %v but labels has length %v", len(batchedOutputDataBboxes), len(batchedOutputDataLabels))
		return nil, fmt.Errorf("Inconsistent batch size for bboxes and labels")
	}

	return DetectionOutput{
		Boxes:  batchedOutputDataBboxes,
		Labels: batchedOutputDataLabels,
	}, nil
}

func postProcessClassification(modelInferResponse *inferenceserver.ModelInferResponse, outputName string) (interface{}, error) {
	outputTensor, rawOutputContent, err := GetOutputFromInferResponse(outputName, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("Unable to find inference output")
	}
	if rawOutputContent == nil {
		return nil, fmt.Errorf("Unable to find output content")
	}

	outputData := DeserializeBytesTensor(rawOutputContent, outputTensor.Shape[0]*outputTensor.Shape[1])

	return outputData, nil
}

func (ts *tritonService) PostProcess(inferResponse *inferenceserver.ModelInferResponse, modelMetadata *inferenceserver.ModelMetadataResponse, task model.Model_Task) (interface{}, error) {
	var (
		outputs interface{}
		err     error
	)

	switch task {
	case model.Model_TASK_CLASSIFICATION:
		outputs, err = postProcessClassification(inferResponse, modelMetadata.Outputs[0].Name)
		if err != nil {
			return nil, fmt.Errorf("Unable to post-process classification output: %w", err)
		}
	case model.Model_TASK_DETECTION:
		outputs, err = postProcessDetection(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name)
		if err != nil {
			return nil, fmt.Errorf("Unable to post-process detection output: %w", err)
		}
	default:
		return inferResponse, nil
	}

	return outputs, nil
}

func (ts *tritonService) LoadModelRequest(modelName string) (*inferenceserver.RepositoryModelLoadResponse, error) {
	fmt.Println(">LoadModelRequestLoadModelRequest")
	// Create context for our request with 60 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create status request for a given model
	loadModelRequest := inferenceserver.RepositoryModelLoadRequest{
		RepositoryName: "",
		ModelName:      modelName,
	}
	// Submit loadModelRequest request to server
	return ts.tritonClient.RepositoryModelLoad(ctx, &loadModelRequest)
}

func (ts *tritonService) UnloadModelRequest(modelName string) (*inferenceserver.RepositoryModelUnloadResponse, error) {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create status request for a given model
	unloadModelRequest := inferenceserver.RepositoryModelUnloadRequest{
		RepositoryName: "",
		ModelName:      modelName,
	}
	// Submit loadModelRequest request to server
	return ts.tritonClient.RepositoryModelUnload(ctx, &unloadModelRequest)
}

func (ts *tritonService) ListModelsRequest() *inferenceserver.RepositoryIndexResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

func (ts *tritonService) IsTritonServerReady() bool {
	serverLiveResponse := ts.ServerLiveRequest()
	if serverLiveResponse == nil {
		return false
	}
	fmt.Printf("Triton Health - Live: %v\n", serverLiveResponse.Live)
	if !serverLiveResponse.Live {
		return false
	}

	serverReadyResponse := ts.ServerReadyRequest()
	fmt.Printf("Triton Health - Ready: %v\n", serverReadyResponse.Ready)
	return serverReadyResponse.Ready
}
