package ray

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"

	"github.com/instill-ai/model-backend/pkg/ray/rayserver"
	// "github.com/instill-ai/model-backend/pkg/ray"
	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
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

func postProcessTextToImage(modelInferResponse *rayserver.RayServiceCallResponse, outputNameImages string, task commonPB.Task) (any, error) {
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
	case commonPB.Task_TASK_IMAGE_TO_IMAGE:
		return ImageToImageOutput{
			Images: batchedOutputDataImages,
		}, nil
	default:
		return TextToImageOutput{
			Images: batchedOutputDataImages,
		}, nil
	}
}

func postProcessTextGeneration(modelInferResponse *rayserver.RayServiceCallResponse, outputNameTexts string, task commonPB.Task) (any, error) {
	outputTensorTexts, rawOutputContentTexts, err := GetOutputFromInferResponse(outputNameTexts, modelInferResponse)
	if err != nil {
		return nil, fmt.Errorf("unable to find inference output for generated texts")
	}
	if outputTensorTexts == nil {
		return nil, fmt.Errorf("unable to find output content for generated texts")
	}
	outputTexts := DeserializeBytesTensor(rawOutputContentTexts, outputTensorTexts.Shape[0])

	switch task {
	case commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING:
		return VisualQuestionAnsweringOutput{
			Text: outputTexts,
		}, nil
	case commonPB.Task_TASK_TEXT_GENERATION_CHAT:

		return TextGenerationChatOutput{
			Text: outputTexts,
		}, nil
	default:

		return TextGenerationOutput{
			Text: outputTexts,
		}, nil
	}
}
