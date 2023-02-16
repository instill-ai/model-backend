package util

import (
	"google.golang.org/protobuf/encoding/protojson"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

var Tasks = map[string]modelPB.ModelInstance_Task{
	"TASK_CLASSIFICATION":        modelPB.ModelInstance_TASK_CLASSIFICATION,
	"TASK_DETECTION":             modelPB.ModelInstance_TASK_DETECTION,
	"TASK_KEYPOINT":              modelPB.ModelInstance_TASK_KEYPOINT,
	"TASK_OCR":                   modelPB.ModelInstance_TASK_OCR,
	"TASK_INSTANCESEGMENTATION":  modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
	"TASK_INSTANCE_SEGMENTATION": modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
	"TASK_SEMANTIC_SEGMENTATION": modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION,
	"TASK_SEMANTICSEGMENTATION":  modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION,
	"TASK_TEXT_TO_IMAGE":         modelPB.ModelInstance_TASK_TEXT_TO_IMAGE,
	"TASK_TEXTTOIMAGE":           modelPB.ModelInstance_TASK_TEXT_TO_IMAGE,
	"TASK_TEXT_GENERATION":       modelPB.ModelInstance_TASK_TEXT_GENERATION,
	"TASK_TEXTGENERATION":        modelPB.ModelInstance_TASK_TEXT_GENERATION,
}

var Tags = map[string]modelPB.ModelInstance_Task{
	"CLASSIFICATION":        modelPB.ModelInstance_TASK_CLASSIFICATION,
	"DETECTION":             modelPB.ModelInstance_TASK_DETECTION,
	"IMAGE-CLASSIFICATION":  modelPB.ModelInstance_TASK_CLASSIFICATION,
	"IMAGE-DETECTION":       modelPB.ModelInstance_TASK_DETECTION,
	"OBJECT-DETECTION":      modelPB.ModelInstance_TASK_DETECTION,
	"OCR":                   modelPB.ModelInstance_TASK_OCR,
	"INSTANCESEGMENTATION":  modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
	"INSTANCE_SEGMENTATION": modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
	"SEMANTIC_SEGMENTATION": modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION,
	"SEMANTICSEGMENTATION":  modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION,
	"TEXT_TO_IMAGE":         modelPB.ModelInstance_TASK_TEXT_TO_IMAGE,
	"TEXTTOIMAGE":           modelPB.ModelInstance_TASK_TEXT_TO_IMAGE,
	"TEXT_GENERATION":       modelPB.ModelInstance_TASK_TEXT_GENERATION,
	"TEXTGENERATION":        modelPB.ModelInstance_TASK_TEXT_GENERATION,
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

type OperationType string

const (
	OperationTypeCreate      OperationType = "create"
	OperationTypeDeploy      OperationType = "deploy"
	OperationTypeUnDeploy    OperationType = "undeploy"
	OperationTypeHealthCheck OperationType = "healthcheck"
)

const DefaultPageSize = 10

const (
	TEXT_TO_IMAGE_STEPS     = int64(10)
	IMAGE_TO_TEXT_CFG_SCALE = float32(7)
	IMAGE_TO_TEXT_SEED      = int64(1024)
	IMAGE_TO_TEXT_SAMPLES   = int64(1)
)

const (
	TEXT_GENERATION_OUTPUT_LEN = int64(100)
	TEXT_GENERATION_TOP_K      = int64(1)
	TEXT_GENERATION_SEED       = int64(0)
)
