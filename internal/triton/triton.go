// Inspired from https://github.com/triton-inference-server/server/blob/v2.5.0/src/clients/go/grpc_simple_client.go

package triton

import (
	"bytes"
	"context"
	"encoding/binary"
	"log"
	"time"

	"github.com/instill-ai/model-backend/configs"
	"github.com/instill-ai/model-backend/internal/inferenceserver"
	"google.golang.org/grpc"
)

const (
	inputSize  = 16
	outputSize = 16
)

func SerializeBytesTensor(tensor [][]byte) []byte {
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

func DeserializeBytesTensor(encodedTensor []byte, capacity int64) []string {
	arr := make([]string, 0, capacity)
	for i := 0; i < len(encodedTensor); {
		length := int(ReadInt32(encodedTensor[i : i+4]))
		i += 4
		arr = append(arr, string(encodedTensor[i:i+length]))
		i += length
	}
	return arr
}

func DeserializeFloat32Tensor(encodedTensor []byte) []float32 {
	arr := make([]float32, len(encodedTensor)/4)
	for i := 0; i < len(encodedTensor)/4; i++ {
		arr[i] = ReadFloat32(encodedTensor[i*4 : i*4+4])
	}
	return arr
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

func ModelInferRequest(client inferenceserver.GRPCInferenceServiceClient, rawInput [][]byte, modelName string, modelVersion string, modelMetadata *inferenceserver.ModelMetadataResponse) *inferenceserver.ModelInferResponse {
	// Create context for our request with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create request input tensors
	inferInputs := []*inferenceserver.ModelInferRequest_InferInputTensor{
		&inferenceserver.ModelInferRequest_InferInputTensor{
			Name:     modelMetadata.Inputs[0].Name,
			Datatype: modelMetadata.Inputs[0].Datatype,
			Shape:    modelMetadata.Inputs[0].Shape,
		},
	}

	// Create request input output tensors
	inferOutputs := []*inferenceserver.ModelInferRequest_InferRequestedOutputTensor{
		&inferenceserver.ModelInferRequest_InferRequestedOutputTensor{
			Name: modelMetadata.Outputs[0].Name,
		},
	}
	// Create inference request for specific model/version
	modelInferRequest := inferenceserver.ModelInferRequest{
		ModelName:    modelName,
		ModelVersion: modelVersion,
		Inputs:       inferInputs,
		Outputs:      inferOutputs,
	}

	modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, rawInput[0])
	modelInferRequest.RawInputContents = append(modelInferRequest.RawInputContents, rawInput[1])

	// Submit inference request to server
	modelInferResponse, err := client.ModelInfer(ctx, &modelInferRequest)
	if err != nil {
		log.Printf("Error processing InferRequest: %v", err)
	}
	return modelInferResponse
}

// Convert int32 input data into raw bytes (assumes Little Endian)
func Preprocess(inputs [][]int32) [][]byte {
	inputData0 := inputs[0]
	inputData1 := inputs[1]

	var inputBytes0 []byte
	var inputBytes1 []byte
	// Temp variable to hold our converted int32 -> []byte
	bs := make([]byte, 4)
	for i := 0; i < inputSize; i++ {
		binary.LittleEndian.PutUint32(bs, uint32(inputData0[i]))
		inputBytes0 = append(inputBytes0, bs...)
		binary.LittleEndian.PutUint32(bs, uint32(inputData1[i]))
		inputBytes1 = append(inputBytes1, bs...)
	}

	return [][]byte{inputBytes0, inputBytes1}
}

// Convert slice of 4 bytes to int32 (assumes Little Endian)
func readInt32(fourBytes []byte) int32 {
	buf := bytes.NewBuffer(fourBytes)
	var retval int32
	binary.Read(buf, binary.LittleEndian, &retval)
	return retval
}

// Convert output's raw bytes into int32 data (assumes Little Endian)
func Postprocess(inferResponse *inferenceserver.ModelInferResponse) [][]int32 {
	outputBytes0 := inferResponse.RawOutputContents[0]
	outputBytes1 := inferResponse.RawOutputContents[1]

	outputData0 := make([]int32, outputSize)
	outputData1 := make([]int32, outputSize)
	for i := 0; i < outputSize; i++ {
		outputData0[i] = readInt32(outputBytes0[i*4 : i*4+4])
		outputData1[i] = readInt32(outputBytes1[i*4 : i*4+4])
	}
	return [][]int32{outputData0, outputData1}
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
