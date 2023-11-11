package utils

import (
	"google.golang.org/protobuf/encoding/protojson"

	commonPB "github.com/instill-ai/protogen-go/common/task/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/model/model/v1alpha"
)

var Tasks = map[string]commonPB.Task{
	"TASK_CLASSIFICATION":        commonPB.Task_TASK_CLASSIFICATION,
	"TASK_DETECTION":             commonPB.Task_TASK_DETECTION,
	"TASK_KEYPOINT":              commonPB.Task_TASK_KEYPOINT,
	"TASK_OCR":                   commonPB.Task_TASK_OCR,
	"TASK_INSTANCESEGMENTATION":  commonPB.Task_TASK_INSTANCE_SEGMENTATION,
	"TASK_INSTANCE_SEGMENTATION": commonPB.Task_TASK_INSTANCE_SEGMENTATION,
	"TASK_SEMANTIC_SEGMENTATION": commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
	"TASK_SEMANTICSEGMENTATION":  commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
	"TASK_TEXT_TO_IMAGE":         commonPB.Task_TASK_TEXT_TO_IMAGE,
	"TASK_TEXTTOIMAGE":           commonPB.Task_TASK_TEXT_TO_IMAGE,
	"TASK_TEXT_GENERATION":       commonPB.Task_TASK_TEXT_GENERATION,
	"TASK_TEXTGENERATION":        commonPB.Task_TASK_TEXT_GENERATION,
}

var Tags = map[string]commonPB.Task{
	"CLASSIFICATION":        commonPB.Task_TASK_CLASSIFICATION,
	"DETECTION":             commonPB.Task_TASK_DETECTION,
	"IMAGE-CLASSIFICATION":  commonPB.Task_TASK_CLASSIFICATION,
	"IMAGE-DETECTION":       commonPB.Task_TASK_DETECTION,
	"OBJECT-DETECTION":      commonPB.Task_TASK_DETECTION,
	"OCR":                   commonPB.Task_TASK_OCR,
	"INSTANCESEGMENTATION":  commonPB.Task_TASK_INSTANCE_SEGMENTATION,
	"INSTANCE_SEGMENTATION": commonPB.Task_TASK_INSTANCE_SEGMENTATION,
	"SEMANTIC_SEGMENTATION": commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
	"SEMANTICSEGMENTATION":  commonPB.Task_TASK_SEMANTIC_SEGMENTATION,
	"TEXT_TO_IMAGE":         commonPB.Task_TASK_TEXT_TO_IMAGE,
	"TEXTTOIMAGE":           commonPB.Task_TASK_TEXT_TO_IMAGE,
	"TEXT_GENERATION":       commonPB.Task_TASK_TEXT_GENERATION,
	"TEXTGENERATION":        commonPB.Task_TASK_TEXT_GENERATION,
}

var Visibility = map[string]modelPB.Model_Visibility{
	"public":  modelPB.Model_VISIBILITY_PUBLIC,
	"private": modelPB.Model_VISIBILITY_PRIVATE,
}

const (
	_  = iota
	KB = 1 << (10 * iota)
	MB
	GB
	TB
)

const MaxBatchSize int = 32

const DEFAULT_GCP_SERVICE_ACCOUNT_FILE = "https://artifacts.instill.tech/default-service-account.json"

var MarshalOptions protojson.MarshalOptions = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: true,
	UseEnumNumbers:  false,
}

var UnmarshalOptions protojson.UnmarshalOptions = protojson.UnmarshalOptions{
	DiscardUnknown: true,
}

const DefaultPageSize = 10

const (
	TEXT_TO_IMAGE_STEPS     = int32(10)
	IMAGE_TO_TEXT_CFG_SCALE = float32(7)
	IMAGE_TO_TEXT_SEED      = int32(1024)
	IMAGE_TO_TEXT_SAMPLES   = int32(1)
)

const (
	TEXT_GENERATION_MAX_NEW_TOKENS = int32(100)
	TEXT_GENERATION_TEMPERATURE    = float32(0.7)
	TEXT_GENERATION_TOP_K          = int32(10)
	TEXT_GENERATION_SEED           = int32(0)
)

const MODEL_CACHE_DIR = "/.cache/models"
const MODEL_CACHE_FILE = "cached_models.json"
