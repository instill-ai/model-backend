// Inspired from https://github.com/triton-inference-server/server/blob/v2.5.0/src/clients/go/grpc_simple_client.go

package triton

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"time"

	"github.com/disintegration/imaging"
	"github.com/instill-ai/model-backend/configs"
	"github.com/instill-ai/model-backend/internal/inferenceserver"
	"google.golang.org/grpc"
)

var TritonClient inferenceserver.GRPCInferenceServiceClient
var connection *grpc.ClientConn

func Init() {
	grpcUri := configs.Config.TritonServer.GrpcUri
	// Connect to gRPC server
	conn, err := grpc.Dial(grpcUri, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Couldn't connect to endpoint %s: %v", grpcUri, err)
	}

	// Create client from gRPC server connection
	connection = conn
	TritonClient = inferenceserver.NewGRPCInferenceServiceClient(conn)
}

func Close() {
	if connection != nil {
		connection.Close()
	}
}

func ServerLiveRequest(client inferenceserver.GRPCInferenceServiceClient) *inferenceserver.ServerLiveResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverLiveRequest := inferenceserver.ServerLiveRequest{}
	// Submit ServerLive request to server
	serverLiveResponse, err := client.ServerLive(ctx, &serverLiveRequest)
	if err != nil {
		log.Printf("Couldn't get server live: %v", err)
	}
	return serverLiveResponse
}

func ServerReadyRequest(client inferenceserver.GRPCInferenceServiceClient) *inferenceserver.ServerReadyResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverReadyRequest := inferenceserver.ServerReadyRequest{}
	// Submit ServerReady request to server
	serverReadyResponse, err := client.ServerReady(ctx, &serverReadyRequest)
	if err != nil {
		log.Printf("Couldn't get server ready: %v", err)
	}
	return serverReadyResponse
}

func ModelMetadataRequest(client inferenceserver.GRPCInferenceServiceClient, modelName string, modelVersion string) *inferenceserver.ModelMetadataResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create status request for a given model
	modelMetadataRequest := inferenceserver.ModelMetadataRequest{
		Name:    modelName,
		Version: modelVersion,
	}
	// Submit modelMetadata request to server
	modelMetadataResponse, err := client.ModelMetadata(ctx, &modelMetadataRequest)
	if err != nil {
		log.Printf("Couldn't get server model metadata: %v", err)
	}
	return modelMetadataResponse
}

func ModelConfigRequest(client inferenceserver.GRPCInferenceServiceClient, modelName string, modelVersion string) *inferenceserver.ModelConfigResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create status request for a given model
	modelConfigRequest := inferenceserver.ModelConfigRequest{
		Name:    modelName,
		Version: modelVersion,
	}
	// Submit modelMetadata request to server
	modelConfigResponse, err := client.ModelConfig(ctx, &modelConfigRequest)
	if err != nil {
		log.Printf("Couldn't get server model config: %v", err)
	}
	return modelConfigResponse
}

func ModelInferRequest(client inferenceserver.GRPCInferenceServiceClient, cvTask CVTask, rawInput [][]byte, modelName string, modelVersion string, modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse) (*inferenceserver.ModelInferResponse, error) {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create request input tensors
	var inferInputs []*inferenceserver.ModelInferRequest_InferInputTensor
	for i := 0; i < len(modelMetadata.Inputs); i++ {
		if modelConfig.Config.Platform == "ensemble" {
			fmt.Println(".   run ensemble")
			inferInputs = append(inferInputs, &inferenceserver.ModelInferRequest_InferInputTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{1, 1},
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
		case Classification:
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
		case Detection:
			inferOutputs = append(inferOutputs, &inferenceserver.ModelInferRequest_InferRequestedOutputTensor{
				Name: modelMetadata.Outputs[i].Name,
			})
		default:
			return nil, fmt.Errorf("Unknown CVTask: %v", cvTask)

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
	modelInferResponse, err := client.ModelInfer(ctx, &modelInferRequest)
	if err != nil {
		log.Printf("Error processing InferRequest: %v", err)
	}

	return modelInferResponse, nil
}

func PreProcess(imageFile string, modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse, cvTask CVTask) (images [][]byte, err error) {
	src, err := imaging.Open(imageFile)
	if err != nil {
		return nil, err
	}

	var img image.Image
	if modelMetadata.Inputs[0].Datatype == "BYTES" { // "BYTES" for "TYPE_STRING", it will be have preprocess model, so no need resize input
		img = src
	} else {
		c, h, w := ParseModel(modelMetadata, modelConfig)
		if c == 1 {
			src = imaging.Grayscale(src)
		}

		img = imaging.Resize(src, int(w), int(h), imaging.Lanczos)
	}

	buff := new(bytes.Buffer)
	err = jpeg.Encode(buff, img, &jpeg.Options{Quality: 100})
	if err != nil {
		return nil, err
	}

	var imgsBytes [][]byte
	imgsBytes = append(imgsBytes, buff.Bytes())

	return imgsBytes, nil
}

func PostProcessDetection(modelInferResponse *inferenceserver.ModelInferResponse, outputNameBboxes string, outputNameLabels string) (interface{}, error) {
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

func PostProcessClassification(modelInferResponse *inferenceserver.ModelInferResponse, outputName string) (interface{}, error) {
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

func PostProcess(inferResponse *inferenceserver.ModelInferResponse, modelMetadata *inferenceserver.ModelMetadataResponse, cvTask CVTask) (interface{}, error) {
	var (
		outputs interface{}
		err     error
	)

	switch cvTask {
	case Classification:
		outputs, err = PostProcessClassification(inferResponse, modelMetadata.Outputs[0].Name)
		if err != nil {
			return nil, fmt.Errorf("Unable to post-process classification output: %w", err)
		}
	case Detection:
		outputs, err = PostProcessDetection(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name)
		if err != nil {
			return nil, fmt.Errorf("Unable to post-process detection output: %w", err)
		}
	default:
		return nil, fmt.Errorf("Unknown CV task %v", cvTask)
	}

	return outputs, nil
}

func LoadModelRequest(client inferenceserver.GRPCInferenceServiceClient, modelName string) *inferenceserver.RepositoryModelLoadResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create status request for a given model
	loadModelRequest := inferenceserver.RepositoryModelLoadRequest{
		RepositoryName: "",
		ModelName:      modelName,
	}
	// Submit loadModelRequest request to server
	loadModelResponse, err := client.RepositoryModelLoad(ctx, &loadModelRequest)
	if err != nil {
		log.Printf("Couldn't load model: %v", err)
	}
	return loadModelResponse
}

func UnloadModelRequest(client inferenceserver.GRPCInferenceServiceClient, modelName string) *inferenceserver.RepositoryModelUnloadResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create status request for a given model
	loadModelRequest := inferenceserver.RepositoryModelUnloadRequest{
		RepositoryName: "",
		ModelName:      modelName,
	}
	// Submit loadModelRequest request to server
	unloadModelResponse, err := client.RepositoryModelUnload(ctx, &loadModelRequest)
	if err != nil {
		log.Printf("Couldn't unload model: %v", err)
	}
	return unloadModelResponse
}

func ListModelsRequest(client inferenceserver.GRPCInferenceServiceClient) *inferenceserver.RepositoryIndexResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create status request for a given model
	listModelsRequest := inferenceserver.RepositoryIndexRequest{
		RepositoryName: "",
	}
	// Submit loadModelRequest request to server
	listModelsResponse, err := client.RepositoryIndex(ctx, &listModelsRequest)
	if err != nil {
		log.Printf("Couldn't list models: %v", err)
	}
	return listModelsResponse
}
