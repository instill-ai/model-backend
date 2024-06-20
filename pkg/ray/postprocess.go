package ray

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/model-backend/pkg/ray/rayserver"
	"github.com/instill-ai/model-backend/pkg/utils"
	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

func postProcessDetection(modelInferResponse *rayserver.RayServiceCallResponse, outputNameBboxes string, outputNameLabels string) (any, error) {
	outputTensorBboxes, rawOutputContentBboxes, err := GetOutputFromInferResponse(outputNameBboxes, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for boxes")
	}
	if rawOutputContentBboxes == nil {
		return nil, fmt.Errorf("unable to find output content for boxes")
	}
	outputTensorLabels, rawOutputContentLabels, err := GetOutputFromInferResponse(outputNameLabels, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for labels")
	}
	if rawOutputContentLabels == nil {
		return nil, fmt.Errorf("unable to find output content for labels")
	}

	outputDataBboxes := DeserializeFloat32Tensor(rawOutputContentBboxes)
	batchedOutputDataBboxes, err := Reshape1DArrayFloat32To3D(outputDataBboxes, outputTensorBboxes.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for boxes")
	}

	outputDataLabels := DeserializeBytesTensor(rawOutputContentLabels, outputTensorBboxes.Shape[0]*outputTensorBboxes.Shape[1])
	batchedOutputDataLabels, err := Reshape1DArrayStringTo2D(outputDataLabels, outputTensorLabels.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for labels")
	}

	if len(batchedOutputDataBboxes) != len(batchedOutputDataLabels) {
		log.Printf("Bboxes output has length %v but labels has length %v", len(batchedOutputDataBboxes), len(batchedOutputDataLabels))
		return nil, fmt.Errorf("inconsistent batch size for bboxes and labels")
	}

	return DetectionOutput{
		Boxes:  batchedOutputDataBboxes,
		Labels: batchedOutputDataLabels,
	}, nil
}

func postProcessOcrWithScore(modelInferResponse *rayserver.RayServiceCallResponse, outputNameBboxes string, outputNameLabels string, outputNameScores string) (any, error) {
	outputTensorBboxes, rawOutputContentBboxes, err := GetOutputFromInferResponse(outputNameBboxes, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for boxes")
	}
	if rawOutputContentBboxes == nil {
		return nil, fmt.Errorf("unable to find output content for boxes")
	}
	outputTensorLabels, rawOutputContentLabels, err := GetOutputFromInferResponse(outputNameLabels, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for labels")
	}
	if rawOutputContentLabels == nil {
		return nil, fmt.Errorf("unable to find output content for labels")
	}
	outputTensorScores, rawOutputContentScores, err := GetOutputFromInferResponse(outputNameScores, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for scores")
	}
	if rawOutputContentScores == nil {
		return nil, fmt.Errorf("unable to find output content for scores")
	}

	outputDataBboxes := DeserializeFloat32Tensor(rawOutputContentBboxes)
	batchedOutputDataBboxes, err := Reshape1DArrayFloat32To3D(outputDataBboxes, outputTensorBboxes.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for boxes")
	}

	outputDataLabels := DeserializeBytesTensor(rawOutputContentLabels, outputTensorLabels.Shape[0]*outputTensorLabels.Shape[1])
	batchedOutputDataLabels, err := Reshape1DArrayStringTo2D(outputDataLabels, outputTensorLabels.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for labels")
	}

	outputDataScores := DeserializeFloat32Tensor(rawOutputContentScores)
	batchedOutputDataScores, err := Reshape1DArrayFloat32To2D(outputDataScores, outputTensorScores.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for labels")
	}

	if len(batchedOutputDataBboxes) != len(batchedOutputDataLabels) || len(batchedOutputDataLabels) != len(batchedOutputDataScores) {
		log.Printf("Bboxes output has length %v but labels has length %v and scores has length %v", len(batchedOutputDataBboxes), len(batchedOutputDataLabels), len(batchedOutputDataScores))
		return nil, fmt.Errorf("inconsistent batch size for bboxes and labels")
	}

	return OcrOutput{
		Boxes:  batchedOutputDataBboxes,
		Texts:  batchedOutputDataLabels,
		Scores: batchedOutputDataScores,
	}, nil
}

func postProcessOcrWithoutScore(modelInferResponse *rayserver.RayServiceCallResponse, outputNameBboxes string, outputNameLabels string) (any, error) {
	outputTensorBboxes, rawOutputContentBboxes, err := GetOutputFromInferResponse(outputNameBboxes, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for boxes")
	}
	if rawOutputContentBboxes == nil {
		return nil, fmt.Errorf("unable to find output content for boxes")
	}
	outputTensorLabels, rawOutputContentLabels, err := GetOutputFromInferResponse(outputNameLabels, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for labels")
	}
	if rawOutputContentLabels == nil {
		return nil, fmt.Errorf("unable to find output content for labels")
	}

	outputDataBboxes := DeserializeFloat32Tensor(rawOutputContentBboxes)
	batchedOutputDataBboxes, err := Reshape1DArrayFloat32To3D(outputDataBboxes, outputTensorBboxes.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for boxes")
	}

	outputDataLabels := DeserializeBytesTensor(rawOutputContentLabels, outputTensorLabels.Shape[0]*outputTensorLabels.Shape[1])
	batchedOutputDataLabels, err := Reshape1DArrayStringTo2D(outputDataLabels, outputTensorLabels.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for labels")
	}

	if len(batchedOutputDataBboxes) != len(batchedOutputDataLabels) {
		log.Printf("Bboxes output has length %v but labels has length %v", len(batchedOutputDataBboxes), len(batchedOutputDataLabels))
		return nil, fmt.Errorf("inconsistent batch size for bboxes and labels")
	}

	var batchedOutputDataScores [][]float32
	for i := range batchedOutputDataLabels {
		var batchedOutputDataScore []float32
		for range batchedOutputDataLabels[i] {
			batchedOutputDataScore = append(batchedOutputDataScore, -1)
		}
		batchedOutputDataScores = append(batchedOutputDataScores, batchedOutputDataScore)
	}

	return OcrOutput{
		Boxes:  batchedOutputDataBboxes,
		Texts:  batchedOutputDataLabels,
		Scores: batchedOutputDataScores,
	}, nil
}

func postProcessClassification(modelInferResponse *rayserver.RayServiceCallResponse, outputName string) (any, error) {
	outputTensor, rawOutputContent, err := GetOutputFromInferResponse(outputName, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output")
	}
	if rawOutputContent == nil {
		return nil, fmt.Errorf("unable to find output content")
	}
	outputData := DeserializeBytesTensor(rawOutputContent, outputTensor.Shape[0]*outputTensor.Shape[1])
	return outputData, nil
}

func postProcessUnspecifiedTask(modelInferResponse *rayserver.RayServiceCallResponse, outputs []*rayserver.ModelMetadataResponse_TensorMetadata) (any, error) {
	var postprocessedOutputs []BatchUnspecifiedTaskOutputs
	for _, output := range outputs {
		outputTensor, rawOutputContent, err := GetOutputFromInferResponse(output.Name, modelInferResponse)
		if err != nil {
			log.Printf("%v", err.Error())
			return nil, fmt.Errorf("unable to find inference output")
		}
		if rawOutputContent == nil {
			return nil, fmt.Errorf("unable to find output content")
		}

		var serializedOutputs []any
		switch output.Datatype {
		case "BYTES":
			if len(outputTensor.Shape) == 1 {
				deserializedRawOutput := DeserializeBytesTensor(rawOutputContent, outputTensor.Shape[0])
				serializedOutputs = append(serializedOutputs, deserializedRawOutput)
			} else {
				deserializedRawOutput := DeserializeBytesTensor(rawOutputContent, outputTensor.Shape[0]*outputTensor.Shape[1])
				reshapedOutputs, _ := Reshape1DArrayStringTo2D(deserializedRawOutput, outputTensor.Shape)
				for _, reshapedOutput := range reshapedOutputs {
					serializedOutputs = append(serializedOutputs, reshapedOutput)
				}
			}
		case "FP32":
			deserializedRawOutput := DeserializeFloat32Tensor(rawOutputContent)
			if len(outputTensor.Shape) == 1 {
				serializedOutputs = append(serializedOutputs, deserializedRawOutput)
			} else if len(outputTensor.Shape) == 2 {
				reshapedOutputs, err := Reshape1DArrayFloat32To2D(deserializedRawOutput, outputTensor.Shape)
				if err != nil {
					return nil, err
				}
				for _, reshapedOutput := range reshapedOutputs {
					serializedOutputs = append(serializedOutputs, reshapedOutput)
				}
			} else if len(outputTensor.Shape) == 3 {
				reshapedOutputs, err := Reshape1DArrayFloat32To3D(deserializedRawOutput, outputTensor.Shape)
				if err != nil {
					return nil, err
				}
				for _, reshapedOutput := range reshapedOutputs {
					serializedOutputs = append(serializedOutputs, reshapedOutput)
				}
			}
		case "INT32":
			deserializedRawOutput := DeserializeInt32Tensor(rawOutputContent)
			if len(outputTensor.Shape) == 1 {
				serializedOutputs = append(serializedOutputs, deserializedRawOutput)
			} else if len(outputTensor.Shape) == 2 {
				reshapedOutputs, err := Reshape1DArrayInt32To2D(deserializedRawOutput, outputTensor.Shape)
				if err != nil {
					return nil, err
				}
				for _, reshapedOutput := range reshapedOutputs {
					serializedOutputs = append(serializedOutputs, reshapedOutput)
				}
			}
		case "STRING":
			deserializedRawOutput := DeserializeBytesTensor(rawOutputContent, outputTensor.Shape[0]*outputTensor.Shape[1])
			reshapedOutputs, err := Reshape1DArrayStringTo2D(deserializedRawOutput, outputTensor.Shape)
			if err != nil {
				return nil, err
			}
			for _, reshapedOutput := range reshapedOutputs {
				serializedOutputs = append(serializedOutputs, reshapedOutput)
			}
		default:
			return nil, fmt.Errorf("unable to decode inference output")
		}
		var shape []int64
		if len(outputTensor.Shape) == 1 {
			shape = outputTensor.Shape
		} else {
			shape = outputTensor.Shape[1:]
		}
		postprocessedOutputs = append(postprocessedOutputs, BatchUnspecifiedTaskOutputs{
			Name:              output.Name,
			Shape:             shape,
			DataType:          output.Datatype,
			SerializedOutputs: serializedOutputs,
		})
	}
	return postprocessedOutputs, nil
}

func postProcessKeypoint(modelInferResponse *rayserver.RayServiceCallResponse, outputNameKeypoints string, outputNameBoxes string, outputNameScores string) (any, error) {
	outputTensorKeypoints, rawOutputContentKeypoints, err := GetOutputFromInferResponse(outputNameKeypoints, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for keypoints")
	}
	if rawOutputContentKeypoints == nil {
		return nil, fmt.Errorf("unable to find output content for keypoints")
	}

	outputTensorBoxes, rawOutputContentBoxes, err := GetOutputFromInferResponse(outputNameBoxes, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for labels")
	}
	if rawOutputContentBoxes == nil {
		return nil, fmt.Errorf("unable to find output content for labels")
	}

	outputTensorScores, rawOutputContentScores, err := GetOutputFromInferResponse(outputNameScores, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for labels")
	}
	if rawOutputContentScores == nil {
		return nil, fmt.Errorf("unable to find output content for labels")
	}

	outputDataKeypoints := DeserializeFloat32Tensor(rawOutputContentKeypoints)
	batchedOutputDataKeypoints, err := Reshape1DArrayFloat32To4D(outputDataKeypoints, outputTensorKeypoints.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for keypoints")
	}

	outputDataBoxes := DeserializeFloat32Tensor(rawOutputContentBoxes)
	batchedOutputDataBoxes, err := Reshape1DArrayFloat32To3D(outputDataBoxes, outputTensorBoxes.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for boxes")
	}

	outputDataScores := DeserializeFloat32Tensor(rawOutputContentScores)
	batchedOutputDataScores, err := Reshape1DArrayFloat32To2D(outputDataScores, outputTensorScores.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for scores")
	}
	if len(batchedOutputDataKeypoints) != len(batchedOutputDataBoxes) || len(batchedOutputDataBoxes) != len(batchedOutputDataScores) {
		log.Printf("Keypoints output has length %v but scores has length %v", len(batchedOutputDataKeypoints), len(batchedOutputDataScores))
		return nil, fmt.Errorf("inconsistent batch size for keypoints and scores")
	}

	return KeypointOutput{
		Keypoints: batchedOutputDataKeypoints,
		Boxes:     batchedOutputDataBoxes,
		Scores:    batchedOutputDataScores,
	}, nil
}

func postProcessInstanceSegmentation(modelInferResponse *rayserver.RayServiceCallResponse, outputNameRles string, outputNameBboxes string, outputNameLabels string, outputNameScores string) (any, error) {
	outputTensorRles, rawOutputContentRles, err := GetOutputFromInferResponse(outputNameRles, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for RLEs")
	}
	if rawOutputContentRles == nil {
		return nil, fmt.Errorf("unable to find output content for RLEs")
	}

	outputTensorBboxes, rawOutputContentBboxes, err := GetOutputFromInferResponse(outputNameBboxes, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for boxes")
	}
	if rawOutputContentBboxes == nil {
		return nil, fmt.Errorf("unable to find output content for boxes")
	}
	outputTensorLabels, rawOutputContentLabels, err := GetOutputFromInferResponse(outputNameLabels, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for labels")
	}
	if rawOutputContentLabels == nil {
		return nil, fmt.Errorf("unable to find output content for labels")
	}

	outputDataLabels := DeserializeBytesTensor(rawOutputContentLabels, outputTensorLabels.Shape[0]*outputTensorLabels.Shape[1])
	batchedOutputDataLabels, err := Reshape1DArrayStringTo2D(outputDataLabels, outputTensorLabels.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for labels")
	}

	outputTensorScores, rawOutputContentScores, err := GetOutputFromInferResponse(outputNameScores, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for scores")
	}
	if rawOutputContentScores == nil {
		return nil, fmt.Errorf("unable to find output content for scores")
	}
	outputDataRles := DeserializeBytesTensor(rawOutputContentRles, outputTensorRles.Shape[0]*outputTensorRles.Shape[1])
	batchedOutputDataRles, err := Reshape1DArrayStringTo2D(outputDataRles, outputTensorRles.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for RLEs")
	}

	outputDataBboxes := DeserializeFloat32Tensor(rawOutputContentBboxes)
	batchedOutputDataBboxes, err := Reshape1DArrayFloat32To3D(outputDataBboxes, outputTensorBboxes.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for boxes")
	}

	outputDataScores := DeserializeFloat32Tensor(rawOutputContentScores)
	batchedOutputDataScores, err := Reshape1DArrayFloat32To2D(outputDataScores, outputTensorScores.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for scores")
	}

	if len(batchedOutputDataBboxes) != len(batchedOutputDataLabels) ||
		len(batchedOutputDataBboxes) != len(batchedOutputDataRles) ||
		len(batchedOutputDataBboxes) != len(batchedOutputDataScores) {
		log.Printf("Rles output has length %v Bboxes output has length %v but labels has length %v scores have length %v",
			len(batchedOutputDataRles), len(batchedOutputDataBboxes), len(batchedOutputDataLabels), len(batchedOutputDataScores))
		return nil, fmt.Errorf("inconsistent batch size for rles, bboxes, labels and scores")
	}

	return InstanceSegmentationOutput{
		Rles:   batchedOutputDataRles,
		Boxes:  batchedOutputDataBboxes,
		Labels: batchedOutputDataLabels,
		Scores: batchedOutputDataScores,
	}, nil
}

func postProcessSemanticSegmentation(modelInferResponse *rayserver.RayServiceCallResponse, outputNameRles string, outputNameCategories string) (any, error) {
	outputTensorRles, rawOutputContentRles, err := GetOutputFromInferResponse(outputNameRles, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for RLEs")
	}
	if rawOutputContentRles == nil {
		return nil, fmt.Errorf("unable to find output content for RLEs")
	}

	outputTensorCategories, rawOutputContentCategories, err := GetOutputFromInferResponse(outputNameCategories, modelInferResponse)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to find inference output for labels")
	}
	if rawOutputContentCategories == nil {
		return nil, fmt.Errorf("unable to find output content for labels")
	}

	outputDataLabels := DeserializeBytesTensor(rawOutputContentCategories, outputTensorCategories.Shape[0]*outputTensorCategories.Shape[1])
	batchedOutputDataCategories, err := Reshape1DArrayStringTo2D(outputDataLabels, outputTensorCategories.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for labels")
	}

	outputDataRles := DeserializeBytesTensor(rawOutputContentRles, outputTensorRles.Shape[0]*outputTensorRles.Shape[1])
	batchedOutputDataRles, err := Reshape1DArrayStringTo2D(outputDataRles, outputTensorRles.Shape)
	if err != nil {
		log.Printf("%v", err.Error())
		return nil, fmt.Errorf("unable to reshape inference output for RLEs")
	}

	if len(batchedOutputDataCategories) != len(batchedOutputDataRles) {
		log.Printf("Rles output has length %v but categories has length %v",
			len(batchedOutputDataCategories), len(batchedOutputDataRles))
		return nil, fmt.Errorf("inconsistent batch size for rles and categories")
	}

	return SemanticSegmentationOutput{
		Rles:       batchedOutputDataRles,
		Categories: batchedOutputDataCategories,
	}, nil
}

func postProcessTextToImage(modelInferResponse *rayserver.RayServiceCallResponse, outputNameImages string, task commonpb.Task) (any, error) {
	outputTensorImages, rawOutputContentImages, err := GetOutputFromInferResponse(outputNameImages, modelInferResponse)
	if err != nil {
		return nil, fmt.Errorf("unable to find inference output for images")
	}
	if outputTensorImages == nil {
		return nil, fmt.Errorf("unable to find output content for images")
	}
	var batchedOutputDataImages [][]string
	batchedOutputDataImages = append(batchedOutputDataImages, []string{}) // single batch support
	var lenSingleImage = len(rawOutputContentImages) / int(outputTensorImages.Shape[0])
	for i := 0; i < int(outputTensorImages.Shape[0]); i++ {
		imgRaw := DeserializeFloat32Tensor(rawOutputContentImages[i*lenSingleImage : (i+1)*lenSingleImage])

		width := int(outputTensorImages.Shape[2])
		height := int(outputTensorImages.Shape[1])
		upLeft := image.Point{0, 0}
		lowRight := image.Point{width, height}

		imgRGBA := image.NewRGBA(image.Rectangle{upLeft, lowRight})
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				imgRGBA.Set(x, y, color.RGBA{uint8(imgRaw[3*(x+width*y)] * 255), uint8(imgRaw[3*(x+width*y)+1] * 255), uint8(imgRaw[3*(x+width*y)+2] * 255), 0xff})
			}
		}

		buff := new(bytes.Buffer)
		err = jpeg.Encode(buff, imgRGBA, &jpeg.Options{Quality: 100})
		if err != nil {
			return nil, fmt.Errorf("jpeg.Encode %w", err)
		}

		base64EncodedStr := base64.StdEncoding.EncodeToString(buff.Bytes())
		batchedOutputDataImages[0] = append(batchedOutputDataImages[0], base64EncodedStr)
	}
	switch task {
	case commonpb.Task_TASK_IMAGE_TO_IMAGE:
		return ImageToImageOutput{
			Images: batchedOutputDataImages,
		}, nil
	default:
		return TextToImageOutput{
			Images: batchedOutputDataImages,
		}, nil
	}
}

func postProcessTextGeneration(modelInferResponse *rayserver.RayServiceCallResponse, outputNameTexts string, task commonpb.Task) (any, error) {
	outputTensorTexts, rawOutputContentTexts, err := GetOutputFromInferResponse(outputNameTexts, modelInferResponse)
	if err != nil {
		return nil, fmt.Errorf("unable to find inference output for generated texts")
	}
	if outputTensorTexts == nil {
		return nil, fmt.Errorf("unable to find output content for generated texts")
	}
	outputTexts := DeserializeBytesTensor(rawOutputContentTexts, outputTensorTexts.Shape[0])

	switch task {
	case commonpb.Task_TASK_VISUAL_QUESTION_ANSWERING:
		return VisualQuestionAnsweringOutput{
			Text: outputTexts,
		}, nil
	case commonpb.Task_TASK_TEXT_GENERATION_CHAT:

		return TextGenerationChatOutput{
			Text: outputTexts,
		}, nil
	default:

		return TextGenerationOutput{
			Text: outputTexts,
		}, nil
	}
}

func PostProcess(inferResponse *rayserver.RayServiceCallResponse, modelMetadata *rayserver.ModelMetadataResponse, task commonpb.Task) (outputs []*modelpb.TaskOutput, err error) {

	var postprocessResponse any

	switch task {
	case commonpb.Task_TASK_CLASSIFICATION:
		postprocessResponse, err = postProcessClassification(inferResponse, modelMetadata.Outputs[0].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process classification output: %w", err)
		}
		clsResponses := postprocessResponse.([]string)
		var clsOutputs []*modelpb.TaskOutput
		for _, clsRes := range clsResponses {
			clsResSplit := strings.Split(clsRes, ":")
			if len(clsResSplit) == 2 {
				score, err := strconv.ParseFloat(clsResSplit[0], 32)
				if err != nil {
					return nil, fmt.Errorf("unable to decode inference output")
				}
				clsOutput := modelpb.TaskOutput{
					Output: &modelpb.TaskOutput_Classification{
						Classification: &modelpb.ClassificationOutput{
							Category: clsResSplit[1],
							Score:    float32(score),
						},
					},
				}
				clsOutputs = append(clsOutputs, &clsOutput)
			} else if len(clsResSplit) == 3 {
				score, err := strconv.ParseFloat(clsResSplit[0], 32)
				if err != nil {
					return nil, fmt.Errorf("unable to decode inference output")
				}
				clsOutput := modelpb.TaskOutput{
					Output: &modelpb.TaskOutput_Classification{
						Classification: &modelpb.ClassificationOutput{
							Category: clsResSplit[2],
							Score:    float32(score),
						},
					},
				}
				clsOutputs = append(clsOutputs, &clsOutput)
			} else {
				return nil, fmt.Errorf("unable to decode inference output")
			}
		}
		if len(clsOutputs) == 0 {
			clsOutputs = append(clsOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_Classification{
					Classification: &modelpb.ClassificationOutput{},
				},
			})
		}
		return clsOutputs, nil
	case commonpb.Task_TASK_DETECTION:
		if len(modelMetadata.Outputs) < 2 {
			return nil, fmt.Errorf("wrong output format of detection task")
		}
		postprocessResponse, err = postProcessDetection(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process detection output: %w", err)
		}
		detResponses := postprocessResponse.(DetectionOutput)
		batchedOutputDataBboxes := detResponses.Boxes
		batchedOutputDataLabels := detResponses.Labels
		var detOutputs []*modelpb.TaskOutput
		for i := range batchedOutputDataBboxes {
			var detOutput = modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_Detection{
					Detection: &modelpb.DetectionOutput{
						Objects: []*modelpb.DetectionObject{},
					},
				},
			}
			for j := range batchedOutputDataBboxes[i] {
				box := batchedOutputDataBboxes[i][j]
				label := batchedOutputDataLabels[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and label "0" for Ray to be able to batch Tensors
				if label != "0" {
					bbObj := &modelpb.DetectionObject{
						Category: label,
						Score:    box[4],
						// Convert x1y1x2y2 to xywh where xy is top-left corner
						BoundingBox: &modelpb.BoundingBox{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2] - box[0],
							Height: box[3] - box[1],
						},
					}
					detOutput.GetDetection().Objects = append(detOutput.GetDetection().Objects, bbObj)
				}
			}
			detOutputs = append(detOutputs, &detOutput)
		}
		if len(detOutputs) == 0 {
			detOutputs = append(detOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_Detection{
					Detection: &modelpb.DetectionOutput{
						Objects: []*modelpb.DetectionObject{},
					},
				},
			})
		}
		return detOutputs, nil
	case commonpb.Task_TASK_KEYPOINT:
		if len(modelMetadata.Outputs) < 3 {
			return nil, fmt.Errorf("wrong output format of keypoint detection task")
		}
		postprocessResponse, err = postProcessKeypoint(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name, modelMetadata.Outputs[2].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process keypoint output: %w", err)
		}
		keypointResponse := postprocessResponse.(KeypointOutput)
		var keypointOutputs []*modelpb.TaskOutput
		for i := range keypointResponse.Keypoints { // batch size
			var keypointObjects []*modelpb.KeypointObject
			for j := range keypointResponse.Keypoints[i] { // n keypoints in one image
				if keypointResponse.Scores[i][j] == -1 { // dummy object for batching to make sure every images have same output shape
					continue
				}
				var keypoints []*modelpb.Keypoint
				points := keypointResponse.Keypoints[i][j]
				for k := range points { // 17 point for each keypoint
					if points[k][0] == -1 && points[k][1] == -1 && points[k][2] == -1 { // dummy output for batching to make sure every images have same output shape
						continue
					}
					keypoints = append(keypoints, &modelpb.Keypoint{
						X: points[k][0],
						Y: points[k][1],
						V: points[k][2],
					})
				}
				keypointObjects = append(keypointObjects, &modelpb.KeypointObject{
					Keypoints: keypoints,
					BoundingBox: &modelpb.BoundingBox{
						Left:   keypointResponse.Boxes[i][j][0],
						Top:    keypointResponse.Boxes[i][j][1],
						Width:  keypointResponse.Boxes[i][j][2],
						Height: keypointResponse.Boxes[i][j][3],
					},
					Score: keypointResponse.Scores[i][j],
				})
			}
			keypointOutputs = append(keypointOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_Keypoint{
					Keypoint: &modelpb.KeypointOutput{
						Objects: keypointObjects,
					},
				},
			})
		}
		if len(keypointOutputs) == 0 {
			keypointOutputs = append(keypointOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_Keypoint{
					Keypoint: &modelpb.KeypointOutput{
						Objects: []*modelpb.KeypointObject{},
					},
				},
			})
		}
		return keypointOutputs, nil
	case commonpb.Task_TASK_OCR:
		if len(modelMetadata.Outputs) < 2 {
			return nil, fmt.Errorf("wrong output format of OCR task")
		}
		switch len(modelMetadata.Outputs) {
		case 2:
			postprocessResponse, err = postProcessOcrWithoutScore(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name)
			if err != nil {
				return nil, fmt.Errorf("unable to post-process detection output: %w", err)
			}
		case 3:
			postprocessResponse, err = postProcessOcrWithScore(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name, modelMetadata.Outputs[2].Name)
			if err != nil {
				return nil, fmt.Errorf("unable to post-process detection output: %w", err)
			}
		}
		ocrResponses := postprocessResponse.(OcrOutput)
		batchedOutputDataBboxes := ocrResponses.Boxes
		batchedOutputDataTexts := ocrResponses.Texts
		batchedOutputDataScores := ocrResponses.Scores
		var ocrOutputs []*modelpb.TaskOutput
		for i := range batchedOutputDataBboxes {
			var ocrOutput = modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_Ocr{
					Ocr: &modelpb.OcrOutput{
						Objects: []*modelpb.OcrObject{},
					},
				},
			}
			for j := range batchedOutputDataBboxes[i] {
				box := batchedOutputDataBboxes[i][j]
				text := batchedOutputDataTexts[i][j]
				score := batchedOutputDataScores[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Ray to be able to batch Tensors
				if text != "" && box[0] != -1 {
					ocrOutput.GetOcr().Objects = append(ocrOutput.GetOcr().Objects, &modelpb.OcrObject{
						BoundingBox: &modelpb.BoundingBox{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2],
							Height: box[3],
						},
						Score: score,
						Text:  text,
					})
				}
			}
			ocrOutputs = append(ocrOutputs, &ocrOutput)
		}
		if len(ocrOutputs) == 0 {
			ocrOutputs = append(ocrOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_Ocr{
					Ocr: &modelpb.OcrOutput{
						Objects: []*modelpb.OcrObject{},
					},
				},
			})
		}
		return ocrOutputs, nil
	case commonpb.Task_TASK_INSTANCE_SEGMENTATION:
		if len(modelMetadata.Outputs) < 4 {
			return nil, fmt.Errorf("wrong output format of instance segmentation task")
		}
		postprocessResponse, err = postProcessInstanceSegmentation(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name, modelMetadata.Outputs[2].Name, modelMetadata.Outputs[3].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process instance segmentation output: %w", err)
		}
		instanceSegmentationResponses := postprocessResponse.(InstanceSegmentationOutput)
		batchedOutputDataRles := instanceSegmentationResponses.Rles
		batchedOutputDataBboxes := instanceSegmentationResponses.Boxes
		batchedOutputDataLabels := instanceSegmentationResponses.Labels
		batchedOutputDataScores := instanceSegmentationResponses.Scores
		var instanceSegmentationOutputs []*modelpb.TaskOutput
		for i := range batchedOutputDataBboxes {
			var instanceSegmentationOutput = modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_InstanceSegmentation{
					InstanceSegmentation: &modelpb.InstanceSegmentationOutput{
						Objects: []*modelpb.InstanceSegmentationObject{},
					},
				},
			}
			for j := range batchedOutputDataBboxes[i] {
				rle := batchedOutputDataRles[i][j]
				box := batchedOutputDataBboxes[i][j]
				label := batchedOutputDataLabels[i][j]
				score := batchedOutputDataScores[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Ray to be able to batch Tensors
				if label != "" && rle != "" {
					instanceSegmentationOutput.GetInstanceSegmentation().Objects = append(instanceSegmentationOutput.GetInstanceSegmentation().Objects, &modelpb.InstanceSegmentationObject{
						Rle: rle,
						BoundingBox: &modelpb.BoundingBox{
							Left:   box[0],
							Top:    box[1],
							Width:  box[2],
							Height: box[3],
						},
						Score:    score,
						Category: label,
					})
				}
			}
			instanceSegmentationOutputs = append(instanceSegmentationOutputs, &instanceSegmentationOutput)
		}
		if len(instanceSegmentationOutputs) == 0 {
			instanceSegmentationOutputs = append(instanceSegmentationOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_InstanceSegmentation{
					InstanceSegmentation: &modelpb.InstanceSegmentationOutput{
						Objects: []*modelpb.InstanceSegmentationObject{},
					},
				},
			})
		}
		return instanceSegmentationOutputs, nil
	case commonpb.Task_TASK_SEMANTIC_SEGMENTATION:
		if len(modelMetadata.Outputs) < 2 {
			return nil, fmt.Errorf("wrong output format of semantic segmentation task")
		}
		postprocessResponse, err = postProcessSemanticSegmentation(inferResponse, modelMetadata.Outputs[0].Name, modelMetadata.Outputs[1].Name)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process semantic segmentation output: %w", err)
		}
		semanticSegmentationResponses := postprocessResponse.(SemanticSegmentationOutput)
		batchedOutputDataRles := semanticSegmentationResponses.Rles
		batchedOutputDataCategories := semanticSegmentationResponses.Categories
		var semanticSegmentationOutputs []*modelpb.TaskOutput
		for i := range batchedOutputDataCategories { // loop over images
			var semanticSegmentationOutput = modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_SemanticSegmentation{
					SemanticSegmentation: &modelpb.SemanticSegmentationOutput{
						Stuffs: []*modelpb.SemanticSegmentationStuff{},
					},
				},
			}
			for j := range batchedOutputDataCategories[i] { // single image
				rle := batchedOutputDataRles[i][j]
				category := batchedOutputDataCategories[i][j]
				// Non-meaningful bboxes were added with coords [-1, -1, -1, -1, -1] and text "" for Ray to be able to batch Tensors
				if category != "" && rle != "" {
					semanticSegmentationOutput.GetSemanticSegmentation().Stuffs = append(semanticSegmentationOutput.GetSemanticSegmentation().Stuffs, &modelpb.SemanticSegmentationStuff{
						Rle:      rle,
						Category: category,
					})
				}
			}
			semanticSegmentationOutputs = append(semanticSegmentationOutputs, &semanticSegmentationOutput)
		}
		if len(semanticSegmentationOutputs) == 0 {
			semanticSegmentationOutputs = append(semanticSegmentationOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_SemanticSegmentation{
					SemanticSegmentation: &modelpb.SemanticSegmentationOutput{
						Stuffs: []*modelpb.SemanticSegmentationStuff{},
					},
				},
			})
		}
		return semanticSegmentationOutputs, nil
	case commonpb.Task_TASK_TEXT_TO_IMAGE:
		postprocessResponse, err = postProcessTextToImage(inferResponse, modelMetadata.Outputs[0].Name, task)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process text to image output: %w", err)
		}
		textToImageResponses := postprocessResponse.(TextToImageOutput)
		batchedOutputDataImages := textToImageResponses.Images
		var textToImageOutputs []*modelpb.TaskOutput
		for i := range batchedOutputDataImages { // loop over images
			var textToImageOutput = modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_TextToImage{
					TextToImage: &modelpb.TextToImageOutput{
						Images: batchedOutputDataImages[i],
					},
				},
			}

			textToImageOutputs = append(textToImageOutputs, &textToImageOutput)
		}
		if len(textToImageOutputs) == 0 {
			textToImageOutputs = append(textToImageOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_TextToImage{
					TextToImage: &modelpb.TextToImageOutput{
						Images: []string{},
					},
				},
			})
		}
		return textToImageOutputs, nil
	case commonpb.Task_TASK_IMAGE_TO_IMAGE:
		postprocessResponse, err = postProcessTextToImage(inferResponse, modelMetadata.Outputs[0].Name, task)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process image to image output: %w", err)
		}
		imageToImageResponses := postprocessResponse.(ImageToImageOutput)
		batchedOutputDataImages := imageToImageResponses.Images
		var imageToImageOutputs []*modelpb.TaskOutput
		for i := range batchedOutputDataImages { // loop over images
			var imageToImageOutput = modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_ImageToImage{
					ImageToImage: &modelpb.ImageToImageOutput{
						Images: batchedOutputDataImages[i],
					},
				},
			}

			imageToImageOutputs = append(imageToImageOutputs, &imageToImageOutput)
		}
		if len(imageToImageOutputs) == 0 {
			imageToImageOutputs = append(imageToImageOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_ImageToImage{
					ImageToImage: &modelpb.ImageToImageOutput{
						Images: []string{},
					},
				},
			})
		}
		return imageToImageOutputs, nil
	case commonpb.Task_TASK_TEXT_GENERATION:
		postprocessResponse, err = postProcessTextGeneration(inferResponse, modelMetadata.Outputs[0].Name, task)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process text generation output: %w", err)
		}
		textGenerationResponses := postprocessResponse.(TextGenerationOutput)
		batchedOutputDataTexts := textGenerationResponses.Text
		var textGenerationOutputs []*modelpb.TaskOutput
		for i := range batchedOutputDataTexts {
			var textGenerationOutput = modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_TextGeneration{
					TextGeneration: &modelpb.TextGenerationOutput{
						Text: batchedOutputDataTexts[i],
					},
				},
			}

			textGenerationOutputs = append(textGenerationOutputs, &textGenerationOutput)
		}
		if len(textGenerationOutputs) == 0 {
			textGenerationOutputs = append(textGenerationOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_TextGeneration{
					TextGeneration: &modelpb.TextGenerationOutput{
						Text: "",
					},
				},
			})
		}
		return textGenerationOutputs, nil
	case commonpb.Task_TASK_VISUAL_QUESTION_ANSWERING:
		postprocessResponse, err = postProcessTextGeneration(inferResponse, modelMetadata.Outputs[0].Name, task)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process visual question answering output: %w", err)
		}
		visualQuestionAnsweringResponses := postprocessResponse.(VisualQuestionAnsweringOutput)
		batchedOutputDataTexts := visualQuestionAnsweringResponses.Text
		var visualQuestionAnsweringOutputs []*modelpb.TaskOutput
		for i := range batchedOutputDataTexts {
			var visualQuestionAnsweringOutput = modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_VisualQuestionAnswering{
					VisualQuestionAnswering: &modelpb.VisualQuestionAnsweringOutput{
						Text: batchedOutputDataTexts[i],
					},
				},
			}

			visualQuestionAnsweringOutputs = append(visualQuestionAnsweringOutputs, &visualQuestionAnsweringOutput)
		}
		if len(visualQuestionAnsweringOutputs) == 0 {
			visualQuestionAnsweringOutputs = append(visualQuestionAnsweringOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_VisualQuestionAnswering{
					VisualQuestionAnswering: &modelpb.VisualQuestionAnsweringOutput{
						Text: "",
					},
				},
			})
		}
		return visualQuestionAnsweringOutputs, nil
	case commonpb.Task_TASK_TEXT_GENERATION_CHAT:
		postprocessResponse, err = postProcessTextGeneration(inferResponse, modelMetadata.Outputs[0].Name, task)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process text to text output: %w", err)
		}
		textGenerationChatResponses := postprocessResponse.(TextGenerationChatOutput)
		batchedOutputDataTexts := textGenerationChatResponses.Text
		var textGenerationChatOutputs []*modelpb.TaskOutput
		for i := range batchedOutputDataTexts {
			var textGenerationChatOutput = modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_TextGenerationChat{
					TextGenerationChat: &modelpb.TextGenerationChatOutput{
						Text: batchedOutputDataTexts[i],
					},
				},
			}

			textGenerationChatOutputs = append(textGenerationChatOutputs, &textGenerationChatOutput)
		}
		if len(textGenerationChatOutputs) == 0 {
			textGenerationChatOutputs = append(textGenerationChatOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_TextGenerationChat{
					TextGenerationChat: &modelpb.TextGenerationChatOutput{
						Text: "",
					},
				},
			})
		}
		return textGenerationChatOutputs, nil
	default:
		postprocessResponse, err = postProcessUnspecifiedTask(inferResponse, modelMetadata.Outputs)
		if err != nil {
			return nil, fmt.Errorf("unable to post-process unspecified output: %w", err)
		}
		outputs := postprocessResponse.([]BatchUnspecifiedTaskOutputs)
		var rawOutputs []*modelpb.TaskOutput
		if len(outputs) == 0 {
			return []*modelpb.TaskOutput{}, nil
		}
		deserializedOutputs := outputs[0].SerializedOutputs
		for i := range deserializedOutputs {
			var singleImageOutput []*structpb.Struct

			for _, output := range outputs {
				unspecifiedOutput := SingleOutputUnspecifiedTaskOutput{
					Name:     output.Name,
					Shape:    output.Shape,
					DataType: output.DataType,
					Data:     output.SerializedOutputs[i],
				}

				var mapOutput map[string]any
				b, err := json.Marshal(unspecifiedOutput)
				if err != nil {
					return nil, err
				}
				if err := json.Unmarshal(b, &mapOutput); err != nil {
					return nil, err
				}
				utils.ConvertAllJSONKeySnakeCase(mapOutput)

				b, err = json.Marshal(mapOutput)
				if err != nil {
					return nil, err
				}
				var structData = &structpb.Struct{}
				err = protojson.Unmarshal(b, structData)

				if err != nil {
					return nil, err
				}
				singleImageOutput = append(singleImageOutput, structData)
			}

			rawOutputs = append(rawOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_Unspecified{
					Unspecified: &modelpb.UnspecifiedOutput{
						RawOutputs: singleImageOutput,
					},
				},
			})
		}
		if len(rawOutputs) == 0 {
			rawOutputs = append(rawOutputs, &modelpb.TaskOutput{
				Output: &modelpb.TaskOutput_Unspecified{
					Unspecified: &modelpb.UnspecifiedOutput{
						RawOutputs: []*structpb.Struct{},
					},
				},
			})
		}
		return rawOutputs, nil
	}
}
