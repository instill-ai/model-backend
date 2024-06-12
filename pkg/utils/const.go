package utils

import (
	"google.golang.org/protobuf/encoding/protojson"

	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

var Tasks = map[string]commonPB.Task{
	"TASK_CLASSIFICATION":            commonPB.Task_TASK_CLASSIFICATION,
	"TASK_DETECTION":                 commonPB.Task_TASK_DETECTION,
	"TASK_KEYPOINT":                  commonPB.Task_TASK_KEYPOINT,
	"TASK_OCR":                       commonPB.Task_TASK_OCR,
	"TASK_INSTANCESEGMENTATION":      commonPB.Task_TASK_INSTANCE_SEGMENTATION,
	"TASK_INSTANCE_SEGMENTATION":     commonPB.Task_TASK_INSTANCE_SEGMENTATION,
	"TASK_SEMANTIC_SEGMENTATION":     commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
	"TASK_SEMANTICSEGMENTATION":      commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
	"TASK_TEXT_TO_IMAGE":             commonPB.Task_TASK_TEXT_TO_IMAGE,
	"TASK_TEXTTOIMAGE":               commonPB.Task_TASK_TEXT_TO_IMAGE,
	"TASK_IMAGE_TO_IMAGE":            commonPB.Task_TASK_IMAGE_TO_IMAGE,
	"TASK_IMAGETOIMAGE":              commonPB.Task_TASK_IMAGE_TO_IMAGE,
	"TASK_TEXT_GENERATION":           commonPB.Task_TASK_TEXT_GENERATION,
	"TASK_TEXTGENERATION":            commonPB.Task_TASK_TEXT_GENERATION,
	"TASK_TEXT_GENERATION_CHAT":      commonPB.Task_TASK_TEXT_GENERATION_CHAT,
	"TASK_TEXTGENERATIONCHAT":        commonPB.Task_TASK_TEXT_GENERATION_CHAT,
	"TASK_VISUAL_QUESTION_ANSWERING": commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING,
	"TASK_VISUALQUESTIONANSWERING":   commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING,
}

var Tags = map[string]commonPB.Task{
	"CLASSIFICATION":                 commonPB.Task_TASK_CLASSIFICATION,
	"DETECTION":                      commonPB.Task_TASK_DETECTION,
	"IMAGE-CLASSIFICATION":           commonPB.Task_TASK_CLASSIFICATION,
	"IMAGE-DETECTION":                commonPB.Task_TASK_DETECTION,
	"OBJECT-DETECTION":               commonPB.Task_TASK_DETECTION,
	"OCR":                            commonPB.Task_TASK_OCR,
	"INSTANCESEGMENTATION":           commonPB.Task_TASK_INSTANCE_SEGMENTATION,
	"INSTANCE_SEGMENTATION":          commonPB.Task_TASK_INSTANCE_SEGMENTATION,
	"SEMANTIC_SEGMENTATION":          commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
	"SEMANTICSEGMENTATION":           commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
	"TEXT_TO_IMAGE":                  commonPB.Task_TASK_TEXT_TO_IMAGE,
	"TEXTTOIMAGE":                    commonPB.Task_TASK_TEXT_TO_IMAGE,
	"IMAGE_TO_IMAGE":                 commonPB.Task_TASK_IMAGE_TO_IMAGE,
	"IMAGETOIMAGE":                   commonPB.Task_TASK_IMAGE_TO_IMAGE,
	"TEXT_GENERATION":                commonPB.Task_TASK_TEXT_GENERATION,
	"TEXTGENERATION":                 commonPB.Task_TASK_TEXT_GENERATION,
	"TEXT_VISUAL_QUESTION_ANSWERING": commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING,
	"TEXTVISUALQUESTIONANSWERING":    commonPB.Task_TASK_VISUAL_QUESTION_ANSWERING,
}

var Visibility = map[string]modelPB.Model_Visibility{
	"public":  modelPB.Model_VISIBILITY_PUBLIC,
	"private": modelPB.Model_VISIBILITY_PRIVATE,
}

const MaxBatchSize int = 32

const DefaultGCPServiceAccountFile = "https://artifacts.instill.tech/default-service-account.json"

var MarshalOptions protojson.MarshalOptions = protojson.MarshalOptions{
	EmitUnpopulated: true,
	UseEnumNumbers:  false,
}

var UnmarshalOptions protojson.UnmarshalOptions = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}

const DefaultPageSize = 10

const (
	ToImageSteps    = int32(10)
	ToImageCFGScale = float32(7)
	ToImageSeed     = int32(1024)
	ToImageSamples  = int32(1)
)

const (
	TextGenerationMaxNewTokens = int32(100)
	TextGenerationTemperature  = float32(0.7)
	TextGenerationTopK         = int32(10)
	TextGenerationSeed         = int32(0)
)
