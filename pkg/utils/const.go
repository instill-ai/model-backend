package utils

import (
	"google.golang.org/protobuf/encoding/protojson"

	commonpb "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelpb "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

var Tasks = map[string]commonpb.Task{
	"TASK_CLASSIFICATION":        commonpb.Task_TASK_CLASSIFICATION,
	"TASK_DETECTION":             commonpb.Task_TASK_DETECTION,
	"TASK_KEYPOINT":              commonpb.Task_TASK_KEYPOINT,
	"TASK_OCR":                   commonpb.Task_TASK_OCR,
	"TASK_INSTANCESEGMENTATION":  commonpb.Task_TASK_INSTANCE_SEGMENTATION,
	"TASK_INSTANCE_SEGMENTATION": commonpb.Task_TASK_INSTANCE_SEGMENTATION,
	"TASK_SEMANTIC_SEGMENTATION": commonpb.Task_TASK_SEMANTIC_SEGMENTATION,
	"TASK_SEMANTICSEGMENTATION":  commonpb.Task_TASK_SEMANTIC_SEGMENTATION,
	"TASK_TEXT_TO_IMAGE":         commonpb.Task_TASK_TEXT_TO_IMAGE,
	"TASK_TEXTTOIMAGE":           commonpb.Task_TASK_TEXT_TO_IMAGE,
	"TASK_IMAGE_TO_IMAGE":        commonpb.Task_TASK_IMAGE_TO_IMAGE,
	"TASK_IMAGETOIMAGE":          commonpb.Task_TASK_IMAGE_TO_IMAGE,
	"TASK_CHAT":                  commonpb.Task_TASK_CHAT,
}

var Tags = map[string]commonpb.Task{
	"CLASSIFICATION":        commonpb.Task_TASK_CLASSIFICATION,
	"DETECTION":             commonpb.Task_TASK_DETECTION,
	"IMAGE-CLASSIFICATION":  commonpb.Task_TASK_CLASSIFICATION,
	"IMAGE-DETECTION":       commonpb.Task_TASK_DETECTION,
	"OBJECT-DETECTION":      commonpb.Task_TASK_DETECTION,
	"OCR":                   commonpb.Task_TASK_OCR,
	"INSTANCESEGMENTATION":  commonpb.Task_TASK_INSTANCE_SEGMENTATION,
	"INSTANCE_SEGMENTATION": commonpb.Task_TASK_INSTANCE_SEGMENTATION,
	"SEMANTIC_SEGMENTATION": commonpb.Task_TASK_SEMANTIC_SEGMENTATION,
	"SEMANTICSEGMENTATION":  commonpb.Task_TASK_SEMANTIC_SEGMENTATION,
	"TEXT_TO_IMAGE":         commonpb.Task_TASK_TEXT_TO_IMAGE,
	"TEXTTOIMAGE":           commonpb.Task_TASK_TEXT_TO_IMAGE,
	"IMAGE_TO_IMAGE":        commonpb.Task_TASK_IMAGE_TO_IMAGE,
	"IMAGETOIMAGE":          commonpb.Task_TASK_IMAGE_TO_IMAGE,
	"CHAT":                  commonpb.Task_TASK_CHAT,
}

var Visibility = map[string]modelpb.Model_Visibility{
	"public":  modelpb.Model_VISIBILITY_PUBLIC,
	"private": modelpb.Model_VISIBILITY_PRIVATE,
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
