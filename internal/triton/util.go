package triton

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"github.com/instill-ai/model-backend/internal/inferenceserver"
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
	if len(encodedTensor) == 0 {
		return []float32{}
	}
	arr := make([]float32, len(encodedTensor)/4)
	for i := 0; i < len(encodedTensor)/4; i++ {
		arr[i] = ReadFloat32(encodedTensor[i*4 : i*4+4])
	}
	return arr
}

// TODO: generalise reshape functions by using interface{} arguments and returned values
func Reshape1DArrayStringTo2D(array []string, shape []int64) ([][]string, error) {
	if len(array) == 0 {
		return [][]string{}, nil
	}

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

func Reshape1DArrayFloat32To3D(array []float32, shape []int64) ([][][]float32, error) {
	if len(array) == 0 {
		return [][][]float32{}, nil
	}

	if len(shape) != 3 {
		return nil, fmt.Errorf("Expected a 3D shape, got %vD shape %v", len(shape), shape)
	}

	var prod int64 = 1
	for _, s := range shape {
		prod *= s
	}
	if prod != int64(len(array)) {
		return nil, fmt.Errorf("Cannot reshape array of length %v into shape %v", len(array), shape)
	}

	res := make([][][]float32, shape[0])
	for i := int64(0); i < shape[0]; i++ {
		res[i] = make([][]float32, shape[1])
		for j := int64(0); j < shape[1]; j++ {
			start := i*shape[1]*shape[2] + j*shape[2]
			end := start + shape[2]
			res[i][j] = array[start:end]
		}

	}

	return res, nil
}

func Reshape1DArrayFloat32To2D(array []float32, shape []int64) ([][]float32, error) {
	if len(array) == 0 {
		return [][]float32{}, nil
	}

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
	res := make([][]float32, shape[0])
	for i := int64(0); i < shape[0]; i++ {
		res[i] = make([]float32, shape[1])
		start := i * shape[1]
		end := start + shape[1]
		res[i] = array[start:end]
	}
	return res, nil
}

func GetOutputFromInferResponse(name string, response *inferenceserver.ModelInferResponse) (*inferenceserver.ModelInferResponse_InferOutputTensor, []byte, error) {
	for idx, output := range response.Outputs {
		if output.Name == name {
			if len(response.RawOutputContents) > 0 {
				return output, response.RawOutputContents[idx], nil
			} else {
				return output, nil, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("Unable to find output named %v", name)
}

func ParseModel(modelMetadata *inferenceserver.ModelMetadataResponse, modelConfig *inferenceserver.ModelConfigResponse) (int64, int64, int64) {
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
