// Inspired from https://github.com/triton-inference-server/server/blob/v2.5.0/src/clients/go/grpc_simple_client.go

package triton

import (
	"bytes"
	"context"
	"encoding/binary"
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

func ReadFloat32(fourBytes []byte) float32 {
	buf := bytes.NewBuffer(fourBytes)
	var result float32
	err := binary.Read(buf, binary.LittleEndian, &result)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

func ReadInt32(fourBytes []byte) int32 {
	buf := bytes.NewBuffer(fourBytes)
	var result int32
	err := binary.Read(buf, binary.LittleEndian, &result)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

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

func _serializeBytesTensor(tensor [][]byte) []byte {
	// Prepend 4-byte length to the input
	// https://github.com/triton-inference-server/server/issues/1100
	// https://github.com/triton-inference-server/server/blob/ffa3d639514a6ba0524bbfef0684238598979c13/src/clients/python/library/tritonclient/utils/__init__.py#L203
	if len(tensor) == 0 {
		return []byte{}
	}

	// Add capacity to avoid memory re-allocation
	res := make([]byte, 0, len(tensor)*(4+len(tensor[0])))
	for _, t := range tensor { // loop over batch
		length := make([]byte, 4)
		binary.LittleEndian.PutUint32(length, uint32(len(t)))
		res = append(res, length...)
		res = append(res, t...)
	}

	return res
}

func _deserializeBytesTensor(encodedTensor []byte) []string {
	var arr []string
	for i := 0; i < len(encodedTensor); {
		length := int(ReadInt32(encodedTensor[i : i+4]))
		i += 4
		arr = append(arr, string(encodedTensor[i:i+length]))
		i += length
	}
	return arr
}

func _reshape1DArrayStringTo2D(array []string, shape []int64) ([][]string, error) {
	if len(shape) != 2 {
		return nil, fmt.Errorf("Expected a 2D shape, got %vD shape %v", len(shape), shape)
	}

	var prod int64 = 1
	for _, s := range shape {
		prod *= s
	}
	if prod != int64(len(array)) {
		return nil, fmt.Errorf("Cannot reshape array of length %v into shape %v", len(array), shape)
	}

	res := make([][]string, shape[0])
	for i := int64(0); i < shape[0]; i++ {
		res[i] = array[i*shape[1] : (i+1)*shape[1]]
	}

	return res, nil
}

func ModelInferRequest(client inferenceserver.GRPCInferenceServiceClient, modelType string, rawInput [][]byte, modelName string, modelVersion string, modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse) *inferenceserver.ModelInferResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create request input tensors
	var inferInputs []*inferenceserver.ModelInferRequest_InferInputTensor
	for i := 0; i < len(modelMetadata.Inputs); i++ {
		if modelConfig.Config.Platform == "ensemble" {
			inferInputs = append(inferInputs, &inferenceserver.ModelInferRequest_InferInputTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{1, 1},
			})
		}
	}

	// Create request input output tensors
	var inferOutputs []*inferenceserver.ModelInferRequest_InferRequestedOutputTensor
	for i := 0; i < len(modelMetadata.Outputs); i++ {
		if modelType == "classification" {
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
		} else {
			//TODO object detection, segmentation
		}
	}

	// Create inference request for specific model/version
	modelInferRequest := inferenceserver.ModelInferRequest{
		ModelName:    modelName,
		ModelVersion: modelVersion,
		Inputs:       inferInputs,
		Outputs:      inferOutputs,
	}
	fmt.Println(">>>> checkccccccc ", _serializeBytesTensor(rawInput)[0], _serializeBytesTensor(rawInput)[10], _serializeBytesTensor(rawInput)[20], _serializeBytesTensor(rawInput)[100])
	modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, _serializeBytesTensor(rawInput))

	// Submit inference request to server
	modelInferResponse, err := client.ModelInfer(ctx, &modelInferRequest)
	if err != nil {
		log.Printf("Error processing InferRequest: %v", err)
	}
	return modelInferResponse
}

func _parseModel(modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse) (int64, int64, int64) {
	input_batch_dim := modelConfig.Config.MaxBatchSize
	var c int64
	var h int64
	var w int64
	if modelConfig.Config.Input[0].Format == 1 { //Format::FORMAT_NHWC = 1
		if input_batch_dim > 0 {
			h = modelMetadata.Inputs[0].Shape[1]
			w = modelMetadata.Inputs[0].Shape[2]
			c = modelMetadata.Inputs[0].Shape[3]
		} else {
			h = modelMetadata.Inputs[0].Shape[0]
			w = modelMetadata.Inputs[0].Shape[1]
			c = modelMetadata.Inputs[0].Shape[2]
		}
	} else {
		if input_batch_dim > 0 {
			c = modelMetadata.Inputs[0].Shape[1]
			h = modelMetadata.Inputs[0].Shape[2]
			w = modelMetadata.Inputs[0].Shape[3]
		} else {
			c = modelMetadata.Inputs[0].Shape[0]
			h = modelMetadata.Inputs[0].Shape[1]
			w = modelMetadata.Inputs[0].Shape[2]
		}
	}
	return c, h, w
}

func Preprocess(modelType string, imageFile string, modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse) (images [][]byte, err error) {
	src, err := imaging.Open(imageFile)
	if err != nil {
		return nil, err
	}

	var img image.Image
	if modelMetadata.Inputs[0].Datatype == "BYTES" { // "BYTES" for "TYPE_STRING", it will be have preprocess model, so no need resize input
		img = src
	} else {
		c, h, w := _parseModel(modelMetadata, modelConfig)
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

// Convert output's raw bytes into int32 data (assumes Little Endian)
func Postprocess(modelType string, inferResponse *inferenceserver.ModelInferResponse, modelMetadata *inferenceserver.ModelMetadataResponse) []string {
	// imageShape := modelMetadata.Inputs[0].Shape
	fmt.Println(">>>>inferResponse.RawOutputContents[0] ", inferResponse.RawOutputContents[0])
	outputData := _deserializeBytesTensor(inferResponse.RawOutputContents[0])
	// fmt.Println(">>>>outputData ", outputData)
	// batchedOutputData, _ := _reshape1DArrayStringTo2D(outputData, imageShape)
	// fmt.Println(">>>>batchedOutputData ", batchedOutputData)
	return outputData
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
