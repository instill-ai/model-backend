package ray

import (
	"encoding/binary"
	"math"

	"github.com/instill-ai/model-backend/pkg/ray/rayserver"

	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
)

func PreProcess(modelName string, version string, inferInput InferInput, task commonPB.Task, modelMetadata *rayserver.ModelMetadataResponse) (modelInferRequest *rayserver.RayServiceCallRequest) {
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
			inferInputs = append(inferInputs, &rayserver.InferTensor{
				Name:     modelMetadata.Inputs[i].Name,
				Datatype: modelMetadata.Inputs[i].Datatype,
				Shape:    []int64{1},
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
	modelInferRequest = &rayserver.RayServiceCallRequest{
		ModelName:    modelName,
		ModelVersion: version,
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

	return modelInferRequest
}
